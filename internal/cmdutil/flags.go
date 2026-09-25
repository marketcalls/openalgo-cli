package cmdutil

import (
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/marketcalls/openalgo-cli/internal/api"
	"github.com/spf13/cobra"
)

// FlagOpts configures RegisterFlags behavior. All map keys use OAS names.
type FlagOpts struct {
	Defaults map[string]string // OAS name -> default value override
}

// RegisterFlags registers CLI flags from generated FlagDef definitions.
// opName is stored as a Cobra annotation so --schema can look it up at
// runtime. The Args validator is wrapped to allow --schema to bypass checks.
func RegisterFlags(cmd *cobra.Command, defs []api.FlagDef, opName string, opts *FlagOpts) {
	if opName != "" {
		if cmd.Annotations == nil {
			cmd.Annotations = map[string]string{}
		}
		cmd.Annotations["op"] = opName

		if origArgs := cmd.Args; origArgs != nil {
			cmd.Args = func(cmd *cobra.Command, args []string) error {
				if v, _ := cmd.Flags().GetBool("schema"); v {
					return nil
				}
				return origArgs(cmd, args)
			}
		}
	}

	// Keep the generated order (required flags first) instead of cobra's
	// alphabetical sort, so --help leads with what must be passed.
	cmd.Flags().SortFlags = false

	for _, d := range defs {
		name := d.Name
		defaultVal := d.Default
		overridden := false
		if opts != nil {
			if def, ok := opts.Defaults[d.OASName]; ok {
				defaultVal, overridden = def, true
			}
		}
		// A spec default that the CLI does not send is applied by the
		// server. Showing it as a cobra default would suggest the value is
		// sent, so it goes into the usage text instead.
		serverDefault := ""
		if defaultVal != "" && !d.CLIDefault && !d.Required && !overridden {
			serverDefault, defaultVal = defaultVal, ""
		}
		usage := FlagUsage(d, serverDefault)

		switch d.Type {
		case "bool":
			cmd.Flags().Bool(name, defaultVal == "true", usage)
		case "int":
			defInt := 0
			if defaultVal != "" {
				var err error
				defInt, err = strconv.Atoi(defaultVal)
				if err != nil {
					log.Fatalf("RegisterFlags: invalid int default %q for flag %q: %v", defaultVal, name, err)
				}
			}
			cmd.Flags().Int(name, defInt, usage)
		case "number", "json", "array", "object":
			// Registered as strings so the exact text reaches the request:
			// numbers are sent as JSON numbers without float rounding, and
			// JSON values are parsed (or read from @file / stdin) at run time
			// by the command, where errors can be reported cleanly.
			cmd.Flags().String(name, defaultVal, usage)
		default:
			cmd.Flags().String(name, defaultVal, usage)
		}

		if len(d.Completions) > 0 {
			_ = cmd.RegisterFlagCompletionFunc(name, cobra.FixedCompletions(d.Completions, cobra.ShellCompDirectiveNoFileComp))
		}
	}
}

// typeString is the FlagDef.Type of plain string flags, the only kind whose
// enum values are listed and checked.
const typeString = "string"

// FlagUsage builds the --help text for a generated flag: the description,
// then "(required)", the allowed values for an enum, and the default the
// server applies when the flag is omitted.
func FlagUsage(d api.FlagDef, serverDefault string) string {
	usage := d.Description
	var notes []string
	if d.Required && !d.CLIDefault {
		notes = append(notes, "required")
	}
	if len(d.Completions) > 0 && d.Type == typeString {
		notes = append(notes, "one of: "+strings.Join(d.Completions, ", "))
	}
	if serverDefault != "" {
		notes = append(notes, "server default: "+serverDefault)
	}
	if len(notes) > 0 {
		usage += " (" + strings.Join(notes, "; ") + ")"
	}
	return usage
}

// CheckEnum reports an error when a string flag with enum values was given a
// value outside them. Case is ignored because OpenAlgo accepts some values
// in either case (e.g. ce/pe); the server stays the final judge.
func CheckEnum(cmd *cobra.Command, d api.FlagDef) error {
	if d.Type != typeString || len(d.Completions) == 0 || !cmd.Flags().Changed(d.Name) {
		return nil
	}
	v := Str(cmd, d.Name)
	for _, c := range d.Completions {
		if strings.EqualFold(v, c) {
			return nil
		}
	}
	return fmt.Errorf("--%s: %q is not one of: %s", d.Name, v, strings.Join(d.Completions, ", "))
}

// CanonicalEnum returns the completion value that matches v ignoring case,
// or v unchanged when none does. CheckEnum accepts any case, but the server
// validates case-sensitively, so the canonical spelling is what gets sent.
func CanonicalEnum(v string, completions []string) string {
	for _, c := range completions {
		if strings.EqualFold(v, c) {
			return c
		}
	}
	return v
}

func Str(cmd *cobra.Command, name string) string {
	v, _ := cmd.Flags().GetString(name)
	return v
}

func RequireStr(cmd *cobra.Command, name string) (string, error) {
	v := Str(cmd, name)
	if v == "" {
		return "", fmt.Errorf("--%s is required (see '%s --help' for examples)", name, cmd.CommandPath())
	}
	return v, nil
}

func RequireAll(cmd *cobra.Command, names ...string) error {
	var missing []string
	for _, n := range names {
		if !isSet(cmd, n) {
			missing = append(missing, "--"+n)
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("%s required (see '%s --help' for examples)", strings.Join(missing, ", "), cmd.CommandPath())
	}
	return nil
}

// isSet reports whether a required flag has a value. String flags need a
// non-empty value (a default counts); int and bool flags have no "empty"
// state, so they must be set explicitly or carry a non-zero default.
func isSet(cmd *cobra.Command, name string) bool {
	f := cmd.Flags().Lookup(name)
	if f == nil {
		return false
	}
	if f.Value.Type() == "string" {
		return f.Value.String() != ""
	}
	return f.Changed || (f.DefValue != "" && f.DefValue != "0" && f.DefValue != "false")
}

func Bool(cmd *cobra.Command, name string) bool {
	v, _ := cmd.Flags().GetBool(name)
	return v
}

func Int(cmd *cobra.Command, name string) int {
	v, _ := cmd.Flags().GetInt(name)
	return v
}

func Changed(cmd *cobra.Command, name string) bool {
	return cmd.Flags().Changed(name)
}
