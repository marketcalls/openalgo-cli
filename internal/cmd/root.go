package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/marketcalls/openalgo-cli/internal/api"
	"github.com/marketcalls/openalgo-cli/internal/client"
	"github.com/marketcalls/openalgo-cli/internal/config"
	"github.com/marketcalls/openalgo-cli/internal/output"
	"github.com/spf13/cobra"
)

const exitAPIError = 1

// boolFalse is the string form of a false boolean flag or env value.
const boolFalse = "false"

var (
	version        = "dev"
	cfg            *config.Resolved
	apiClient      *client.Client
	openalgoClient *api.Client
	csvFlag        bool
	jqFlag         string
	quietFlag      bool
	verboseFlag    bool
	debugFlag      bool
	traceFlag      bool
	schemaFlag     bool
	versionFlag    bool
	profileFlag    string
	timeoutFlag    int
)

func SetVersion(v string) {
	version = v
	client.Version = v
}

func Root() *cobra.Command {
	return rootCmd
}

func Execute() error {
	err := rootCmd.Execute()
	if err != nil {
		var apiErr *client.APIError
		if errors.As(err, &apiErr) {
			printJSONError(apiErr)
			os.Exit(apiErr.ExitCode())
		}

		msg, hint := splitHint(err.Error())
		printJSONError(client.NewError(msg, hint))
		var authErr *authRequiredError
		if errors.As(err, &authErr) {
			os.Exit(client.ExitAuthError)
		}
		os.Exit(exitAPIError)
	}
	return nil
}

// authRequiredError marks a missing API key so Execute exits with the auth
// error code (2) rather than the general one.
type authRequiredError struct{ err error }

func (e *authRequiredError) Error() string { return e.err.Error() }
func (e *authRequiredError) Unwrap() error { return e.err }

// splitHint separates a trailing "\nHint: ..." from an error message so the
// JSON error carries it in the hint field.
func splitHint(msg string) (string, string) {
	if i := strings.Index(msg, "\nHint: "); i >= 0 {
		return msg[:i], strings.TrimSpace(msg[i+len("\nHint: "):])
	}
	return msg, ""
}

// printJSONError writes the error contract to stderr:
// {"error", "status", "hint", "method", "path"}. status is the HTTP status
// code, or 0 when the request never got a response.
func printJSONError(apiErr *client.APIError) {
	m := map[string]any{
		"error":  apiErr.Message,
		"status": apiErr.StatusCode,
		"hint":   apiErr.Hint(),
	}
	if apiErr.Method != "" {
		m["method"] = apiErr.Method
	}
	if apiErr.Path != "" {
		m["path"] = apiErr.Path
	}
	if apiErr.RequestID != "" {
		m["request_id"] = apiErr.RequestID
	}
	enc := json.NewEncoder(os.Stderr)
	enc.SetIndent("", "  ")
	_ = enc.Encode(m)
}

var versionCmd = &cobra.Command{
	Use:     "version",
	Short:   "Print the version of openalgo CLI",
	Long:    `Print the version of openalgo CLI.`,
	Example: `  openalgo version`,
	RunE: func(cmd *cobra.Command, args []string) error {
		_, err := fmt.Fprintln(cmd.OutOrStdout(), version)
		return err
	},
}

var rootCmd = &cobra.Command{
	Use:   "openalgo",
	Short: "CLI for the OpenAlgo trading API",
	Long: `Place and manage orders, read positions, funds and market data, and stream live
quotes from a self-hosted OpenAlgo server, from the command line.

Get started:  openalgo profile login
To update:    openalgo update`,
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		if versionFlag {
			_, err := fmt.Fprintln(cmd.OutOrStdout(), version)
			return err
		}
		if ha, _ := cmd.Flags().GetBool("help-all"); ha {
			printCommandTree(cmd.OutOrStdout(), cmd, 0)
			return nil
		}
		shouldWarn := cfg != nil && envShadowsProfile(cfg.ProfileName)
		if shouldWarn {
			warnEnvShadowsProfile(cfg.ProfileName, "")
			fmt.Fprintln(cmd.OutOrStdout())
		}
		if err := cmd.Help(); err != nil {
			return err
		}
		if shouldWarn {
			fmt.Fprintln(cmd.OutOrStdout())
			warnEnvShadowsProfile(cfg.ProfileName, "")
		}
		printUpdateNoticeIfAvailable(cmd.OutOrStdout(), 2*time.Second)
		return nil
	},
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		if cmd.Name() == "help" {
			return nil
		}
		if ha, _ := cmd.Flags().GetBool("help-all"); ha {
			return nil
		}
		if schemaFlag {
			err := printCommandSchema(cmd)
			cmd.RunE = func(*cobra.Command, []string) error { return nil }
			return err
		}
		if cmd.Parent() != nil && cmd.Parent().Name() == "profile" {
			return nil
		}

		if envBool("OPENALGO_QUIET") {
			quietFlag = true
		}
		if quietFlag {
			color.NoColor = true
		}
		if envBool("OPENALGO_VERBOSE") {
			verboseFlag = true
		}
		if envBool("OPENALGO_DEBUG") {
			debugFlag = true
		}
		if envBool("OPENALGO_TRACE") {
			traceFlag = true
		}

		var err error
		outputOverride := ""
		if csvFlag {
			outputOverride = string(output.FormatCSV)
		}
		cfg, err = config.Load(profileFlag, outputOverride)
		if err != nil {
			return err
		}

		if needsAuth(cmd) {
			if err := cfg.Validate(); err != nil {
				return &authRequiredError{err}
			}
			// Tests can pre-inject apiClient to redirect traffic at a mock
			// server; only build a fresh client when nothing's been
			// injected. CLI flags are applied either way so --debug,
			// --verbose, --quiet, --trace and --timeout behave the same
			// under tests as in production.
			if apiClient == nil {
				apiClient = client.New(cfg)
			}
			if openalgoClient == nil || openalgoClient.Raw != apiClient {
				openalgoClient = api.NewClient(apiClient)
			}
			apiClient.Verbose = verboseFlag
			apiClient.Debug = debugFlag
			apiClient.Quiet = quietFlag
			apiClient.Trace = traceFlag
			if timeoutFlag != 30 {
				if timeoutFlag <= 0 {
					return fmt.Errorf("--timeout must be a positive number of seconds")
				}
				apiClient.SetTimeout(time.Duration(timeoutFlag) * time.Second)
			}
		}

		return nil
	},
}

// envBool treats any value other than "", "0", "false" or "no" as true, so
// OPENALGO_QUIET=0 turns the setting off instead of on.
func envBool(name string) bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv(name))) {
	case "", "0", boolFalse, "no", "off":
		return false
	}
	return true
}

func init() {
	rootCmd.PersistentFlags().BoolVar(&csvFlag, "csv", false, "Output as CSV")
	rootCmd.PersistentFlags().StringVar(&jqFlag, "jq", "", "Filter output with a jq expression")
	rootCmd.PersistentFlags().StringVarP(&profileFlag, "profile", "p", "", "Config profile to use")
	rootCmd.PersistentFlags().BoolVarP(&verboseFlag, "verbose", "v", false, "Show HTTP request details on stderr")
	rootCmd.PersistentFlags().BoolVar(&debugFlag, "debug", false, "Show HTTP request/response headers and bodies on stderr (API key redacted)")
	rootCmd.PersistentFlags().BoolVar(&traceFlag, "trace", false, "Show HTTP timing breakdown on stderr (DNS, TLS, TTFB)")
	rootCmd.PersistentFlags().BoolVarP(&quietFlag, "quiet", "q", false, "Suppress non-data output (warnings, hints, color)")
	rootCmd.PersistentFlags().IntVar(&timeoutFlag, "timeout", 30, "HTTP request timeout in seconds")
	rootCmd.PersistentFlags().BoolVar(&schemaFlag, "schema", false, "Show response schema for this command and exit")

	rootCmd.Flags().Bool("help-all", false, "Print full reference for every command")
	rootCmd.Flags().BoolVar(&versionFlag, "version", false, "Print the version of openalgo CLI")

	tradingGroup := &cobra.Group{ID: "trading", Title: "Trading"}
	accountGroup := &cobra.Group{ID: "account", Title: "Account & Market Data"}
	notifyGroup := &cobra.Group{ID: "notify", Title: "Notifications"}
	utilGroup := &cobra.Group{ID: "util", Title: "Utilities"}

	rootCmd.AddGroup(tradingGroup, accountGroup, notifyGroup, utilGroup)
	rootCmd.SetHelpCommandGroupID(utilGroup.ID)
	rootCmd.SetCompletionCommandGroupID(utilGroup.ID)

	addGroup(rootCmd, tradingGroup.ID, orderCmd, gttCmd, positionCmd, optionCmd, strategyCmd)
	addGroup(rootCmd, accountGroup.ID, accountCmd, dataCmd, symbolCmd, marketCmd, analyzerCmd)
	addGroup(rootCmd, notifyGroup.ID, telegramCmd, whatsappCmd)
	addGroup(rootCmd, utilGroup.ID, chartCmd, pingCmd, profileCmd, apiCmd, doctorCmd, updateCmd, versionCmd)
}

func addGroup(parent *cobra.Command, groupID string, cmds ...*cobra.Command) {
	for _, c := range cmds {
		c.GroupID = groupID
		parent.AddCommand(c)
	}
}

// needsAuth reports whether a command talks to the API. Operational
// commands work without credentials so users can set them up and debug.
// A --dry-run only prints the request body, so it needs no key either.
func needsAuth(cmd *cobra.Command) bool {
	if f := cmd.Flags().Lookup("dry-run"); f != nil && f.Changed && f.Value.String() == "true" {
		return false
	}
	for c := cmd; c != nil; c = c.Parent() {
		// Shell completion (__complete) only lists flag values; it must work
		// before any credentials exist.
		if c.Name() == cobra.ShellCompRequestCmd || c.Name() == cobra.ShellCompNoDescRequestCmd {
			return false
		}
		if c.Parent() != nil && c.Parent().Parent() == nil {
			switch c.Name() {
			case "version", "help", "completion", "update", "doctor", "profile":
				return false
			}
		}
	}
	return cmd.Parent() != nil
}

func getOutput() output.Format {
	if csvFlag {
		return output.FormatCSV
	}
	if cfg != nil {
		return output.Format(cfg.Output)
	}
	return output.FormatJSON
}

// renderData is the unified output pipeline: jq filter -> format render.
//
// Non-JSON bodies (instruments --format csv, data ticker --format txt) are
// written to stdout verbatim. With --csv and no --jq, the rows at the op's
// RowsPath (e.g. "data" or "data.orders") are rendered instead of the
// whole envelope.
func renderData(cmd *cobra.Command, data any) error {
	w := cmd.OutOrStdout()
	if raw, ok := data.(json.RawMessage); ok && len(raw) > 0 && !json.Valid(raw) {
		if jqFlag != "" {
			return fmt.Errorf("--jq cannot be applied: the response is not JSON")
		}
		return writeVerbatim(w, raw)
	}

	if jqFlag != "" {
		var err error
		data, err = output.ApplyJQ(data, jqFlag)
		if err != nil {
			return err
		}
	}
	if getOutput() == output.FormatCSV {
		var headers []string
		if jqFlag == "" {
			if op, ok := opForCommand(cmd); ok && op.RowsPath != "" {
				headers = rowHeaders(op.Response, op.RowsPath)
				// A response without the rows field (e.g. an unexpected
				// shape) is rendered whole rather than as empty output.
				if rows := rowsAtPath(data, op.RowsPath); rows != nil || len(headers) > 0 {
					data = rows
				}
			}
		}
		return output.CSVWithHeaders(w, data, headers)
	}
	return output.Render(w, getOutput(), data)
}

func writeVerbatim(w io.Writer, raw []byte) error {
	if _, err := w.Write(raw); err != nil {
		return err
	}
	if raw[len(raw)-1] != '\n' {
		_, err := io.WriteString(w, "\n")
		return err
	}
	return nil
}

func opForCommand(cmd *cobra.Command) (api.Op, bool) {
	if cmd == nil || cmd.Annotations["op"] == "" {
		return api.Op{}, false
	}
	return api.OpByName(cmd.Annotations["op"])
}

// rowsAtPath walks a dotted path ("data.orders") into a decoded response
// and returns the value found there, or nil when the path is absent.
// Numbers are decoded as json.Number so CSV cells keep the server's text.
func rowsAtPath(data any, path string) any {
	v, err := output.Normalize(data)
	if err != nil {
		return data
	}
	for _, key := range strings.Split(path, ".") {
		m, ok := v.(map[string]any)
		if !ok {
			return nil
		}
		v = m[key]
	}
	return v
}

// rowHeaders returns the CSV column names for the rows at path in the
// response schema: the item fields of an array of objects (nested objects
// expanded into dotted names such as data.ltp), or the leaf name itself
// when the path holds an array of plain values (e.g. expiry dates).
func rowHeaders(fields []api.ResponseField, path string) []string {
	keys := strings.Split(path, ".")
	var node *api.ResponseField
	for _, key := range keys {
		found := false
		for i := range fields {
			if fields[i].Name == key {
				node, fields, found = &fields[i], fields[i].Fields, true
				break
			}
		}
		if !found {
			return nil
		}
	}
	if len(fields) == 0 {
		if node != nil && strings.HasPrefix(node.Type, "[]") {
			return []string{node.Name}
		}
		return nil
	}
	return fieldNames("", fields)
}

func fieldNames(prefix string, fields []api.ResponseField) []string {
	var names []string
	for _, f := range fields {
		name := prefix + f.Name
		if len(f.Fields) > 0 && strings.HasPrefix(f.Type, "object") {
			names = append(names, fieldNames(name+".", f.Fields)...)
			continue
		}
		names = append(names, name)
	}
	return names
}
