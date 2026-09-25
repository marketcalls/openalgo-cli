package cmd

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"slices"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/marketcalls/openalgo-cli/internal/client"
	"github.com/marketcalls/openalgo-cli/internal/cmdutil"
	"github.com/marketcalls/openalgo-cli/internal/config"
	"github.com/marketcalls/openalgo-cli/internal/useragent"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var profileCmd = &cobra.Command{
	Use:   "profile",
	Short: "Manage connection profiles",
	Long: `Manage connection profiles. A profile stores the OpenAlgo server address and
the API key issued by that server, in ~/.config/openalgo/profiles/<name>.yaml
(file mode 0600).`,
}

var profileLoginCmd = &cobra.Command{
	Use:   "login",
	Short: "Save an OpenAlgo host and API key as a profile",
	Long: `Save an OpenAlgo host and API key as a profile and make it the default.

Prompts for the host (default http://127.0.0.1:5000) and the API key when
they are not given as flags; the key is read without echo. The key is
checked against the server with POST /api/v1/ping unless --no-validate is
set. Generate the key under API Key in the OpenAlgo dashboard.

The WebSocket URL used by ` + "`openalgo stream`" + ` is derived from the host; pass
--ws-url to store an override in the profile (--ws-url "" clears it).`,
	Example: `  openalgo profile login
  openalgo profile login --host https://algo.example.com
  openalgo profile login --host http://192.168.1.10:5000 --name office
  openalgo profile login --host https://algo.example.com --ws-url wss://ws.example.com
  openalgo profile login --no-validate`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return loginWithAPIKey(cmd)
	},
}

// loginInput supplies interactive input. Swapped out by tests.
var loginInput io.Reader = os.Stdin

func loginWithAPIKey(cmd *cobra.Command) error {
	host := cmdutil.Str(cmd, "host")
	key := cmdutil.Str(cmd, "key")
	name := cmdutil.Str(cmd, "name")
	noValidate := cmdutil.Bool(cmd, "no-validate")

	if name == "" {
		name = config.DefaultProfileName
	}
	if err := config.ValidateProfileName(name); err != nil {
		return err
	}
	// Profile commands skip the root PersistentPreRunE, so resolve the
	// quiet setting (flag or OPENALGO_QUIET) here.
	quiet := quietFlag || envBool("OPENALGO_QUIET")

	if cmdutil.Changed(cmd, "key") && !quiet {
		fmt.Fprintln(os.Stderr, "Warning: passing the API key via a flag may expose it in shell history.")
		fmt.Fprintln(os.Stderr, "  Run `openalgo profile login` interactively or set OPENALGO_API_KEY instead.")
	}

	reader := bufio.NewReader(loginInput)
	interactive := isTerminal(loginInput)
	// Only prompt for the host on a terminal: when the key is piped in
	// (echo $KEY | openalgo profile login) the first line is the key.
	if interactive && !cmdutil.Changed(cmd, "host") && host == "" {
		fmt.Fprintf(os.Stderr, "OpenAlgo host [%s]: ", config.DefaultHost)
		line, _ := reader.ReadString('\n')
		host = strings.TrimSpace(line)
	}
	host = config.NormalizeHost(host)
	if err := config.ValidateHost(host); err != nil {
		return err
	}

	if key == "" {
		if interactive {
			fmt.Fprint(os.Stderr, "API key: ")
			raw, _ := term.ReadPassword(int(loginInput.(*os.File).Fd()))
			key = string(raw)
			fmt.Fprintln(os.Stderr)
		} else {
			line, _ := reader.ReadString('\n')
			key = line
		}
		key = strings.TrimSpace(key)
	}

	if key == "" {
		return fmt.Errorf("an API key is required\nHint: generate one under API Key in the OpenAlgo dashboard at %s", host)
	}

	broker := ""
	if !noValidate {
		var err error
		broker, err = validateCredentials(host, key)
		if err != nil {
			return err
		}
	}

	p := config.LoadProfileByName(name)
	p.Host = host
	p.APIKey = key
	if cmdutil.Changed(cmd, "ws-url") {
		p.WSURL = strings.TrimSpace(cmdutil.Str(cmd, "ws-url"))
	}
	if err := config.SaveProfile(name, p); err != nil {
		return fmt.Errorf("saving profile: %w", err)
	}

	globalCfg := loadOrCreateGlobal()
	globalCfg.DefaultProfile = name
	if err := config.SaveGlobalConfig(globalCfg); err != nil {
		return fmt.Errorf("saving global config: %w", err)
	}

	if broker != "" {
		color.Green("Logged in to %s (%s, broker: %s)", name, host, broker)
	} else {
		color.Green("Logged in to %s (%s)", name, host)
	}
	if !quiet {
		fmt.Fprintf(os.Stderr, "  Credentials stored in %s/profiles/\n", config.Dir())
		fmt.Fprintln(os.Stderr, "  For CI/automation, use OPENALGO_API_KEY and OPENALGO_HOST env vars instead.")
		warnEnvShadowsProfile(name, "  ")
	}
	return nil
}

// validateCredentials checks the key against POST {host}/api/v1/ping and
// returns the broker name reported by the server.
func validateCredentials(host, key string) (string, error) {
	baseURL := config.BaseURL(host)
	payload, _ := json.Marshal(map[string]string{"apikey": key})
	req, _ := http.NewRequest(http.MethodPost, baseURL+"/ping", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", useragent.Build(version))
	resp, err := (&http.Client{Timeout: 10 * time.Second}).Do(req)
	if err != nil {
		msg := strings.ReplaceAll(err.Error(), key, "[REDACTED]")
		return "", fmt.Errorf("failed to connect to %s: %s\nHint: check that OpenAlgo is running, or use --no-validate to skip the check", host, msg)
	}
	defer func() { _ = resp.Body.Close() }()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))

	var pong struct {
		Status  string `json:"status"`
		Message string `json:"message"`
		Data    struct {
			Broker string `json:"broker"`
		} `json:"data"`
	}
	_ = json.Unmarshal(body, &pong)

	if resp.StatusCode == 401 || resp.StatusCode == 403 {
		// An APIError carrying the status exits with the auth code (2),
		// like any other command rejected for its key.
		apiErr := client.NewError(
			fmt.Sprintf("invalid API key or no active broker session (validated against %s)", host),
			"log in to the broker in the OpenAlgo dashboard and copy the key from API Key, or use --no-validate to skip")
		apiErr.StatusCode = resp.StatusCode
		return "", apiErr
	}
	if resp.StatusCode >= 400 || pong.Status == "error" {
		msg := pong.Message
		if msg == "" {
			msg = http.StatusText(resp.StatusCode)
		}
		return "", fmt.Errorf("unexpected response from %s: HTTP %d: %s\nHint: check that --host points at the OpenAlgo server", host, resp.StatusCode, msg)
	}
	if pong.Status != "success" {
		return "", fmt.Errorf("%s did not answer like an OpenAlgo server\nHint: check that --host points at the OpenAlgo server, e.g. %s", host, config.DefaultHost)
	}
	return pong.Data.Broker, nil
}

// envShadowsProfile reports whether OPENALGO_API_KEY will shadow the named
// profile. The env key beats any stored profile, so if env is set AND the
// profile has a key, the profile is effectively dormant.
func envShadowsProfile(profileName string) bool {
	if os.Getenv("OPENALGO_API_KEY") == "" {
		return false
	}
	return config.LoadProfileByName(profileName).APIKey != ""
}

// warnEnvShadowsProfile prints the shadowing warning when envShadowsProfile
// returns true. Users who don't see it would wonder why "their profile"
// talks to a different server. indent is the leading whitespace for the
// line - pass "  " when rendering inside a nested block (profile login,
// doctor) and "" when rendering flush (bare openalgo help).
func warnEnvShadowsProfile(profileName, indent string) {
	if !envShadowsProfile(profileName) {
		return
	}
	color.Yellow(indent+"Warning: OPENALGO_API_KEY is set in your environment; it will override profile %q on every command.", profileName)
}

var profileLogoutCmd = &cobra.Command{
	Use:   "logout [name]",
	Short: "Remove a profile",
	Example: `  openalgo profile logout
  openalgo profile logout office`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := config.DefaultProfileName
		if len(args) > 0 {
			name = args[0]
		}
		if err := config.DeleteProfile(name); err != nil {
			if os.IsNotExist(err) {
				return config.ProfileNotFoundError(name)
			}
			return err
		}
		w := cmd.OutOrStdout()
		fmt.Fprintf(w, "Removed profile %s.\n", name)

		// Never leave config.yaml pointing at the deleted profile: switch
		// to a remaining one, or clear the default when none is left.
		globalCfg := loadOrCreateGlobal()
		if globalCfg.DefaultProfile == name {
			remaining, _ := config.ListProfiles()
			globalCfg.DefaultProfile = ""
			if len(remaining) > 0 {
				globalCfg.DefaultProfile = remaining[0]
			}
			if err := config.SaveGlobalConfig(globalCfg); err != nil {
				return fmt.Errorf("saving global config: %w", err)
			}
			if globalCfg.DefaultProfile != "" {
				fmt.Fprintf(w, "Active profile is now %s.\n", globalCfg.DefaultProfile)
			} else {
				fmt.Fprintln(w, "No profiles left; run `openalgo profile login` to create one.")
			}
		}
		return nil
	},
}

var profileListCmd = &cobra.Command{
	Use:     "list",
	Short:   "List all profiles",
	Example: `  openalgo profile list`,
	Args:    cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		w := cmd.OutOrStdout()
		profiles, err := config.ListProfiles()
		if err != nil {
			return err
		}
		if len(profiles) == 0 {
			fmt.Fprintln(w, "No profiles configured.")
			fmt.Fprintln(w, "Hint: run `openalgo profile login` to create one")
			return nil
		}

		active := ""
		if resolved, err := config.Load(profileFlag, ""); err == nil {
			active = resolved.ProfileName
		}
		// Pad names so the host column lines up; the active marker has
		// its own leading column and "(active)" goes at the end.
		width := 0
		for _, name := range profiles {
			width = max(width, len(name))
		}
		for _, name := range profiles {
			host := config.NormalizeHost(config.LoadProfileByName(name).Host)
			if name == active {
				color.New(color.FgGreen).Fprintf(w, "* %-*s  %s  (active)\n", width, name, host)
			} else {
				fmt.Fprintf(w, "  %-*s  %s\n", width, name, host)
			}
		}
		if active != "" {
			warnEnvShadowsProfile(active, "")
		}
		return nil
	},
}

var profileSwitchCmd = &cobra.Command{
	Use:   "switch <name>",
	Short: "Switch the active profile",
	Example: `  openalgo profile switch office
  openalgo profile switch default`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		if err := config.ValidateProfileName(name); err != nil {
			return err
		}
		profiles, err := config.ListProfiles()
		if err != nil {
			return err
		}

		if !slices.Contains(profiles, name) {
			available := "(none)"
			if len(profiles) > 0 {
				available = strings.Join(profiles, ", ")
			}
			return fmt.Errorf("profile %q not found\nAvailable: %s\nHint: run `openalgo profile login --name %s` to create it", name, available, name)
		}

		globalCfg := loadOrCreateGlobal()
		globalCfg.DefaultProfile = name
		if err := config.SaveGlobalConfig(globalCfg); err != nil {
			return err
		}

		fmt.Fprintf(cmd.OutOrStdout(), "Switched to %s.\n", name)
		warnEnvShadowsProfile(name, "")
		return nil
	},
}

func init() {
	profileLoginCmd.Flags().String("host", "", "OpenAlgo server URL (default: "+config.DefaultHost+")")
	profileLoginCmd.Flags().String("key", "", "API key (prompted without echo when omitted)")
	profileLoginCmd.Flags().String("name", "", "Profile name (default: "+config.DefaultProfileName+")")
	profileLoginCmd.Flags().String("ws-url", "", "WebSocket URL for stream commands (default: derived from --host)")
	profileLoginCmd.Flags().Bool("no-validate", false, "Skip the API key check against the server")

	profileCmd.AddCommand(profileLoginCmd)
	profileCmd.AddCommand(profileLogoutCmd)
	profileCmd.AddCommand(profileListCmd)
	profileCmd.AddCommand(profileSwitchCmd)
}

// loadOrCreateGlobal returns the stored global settings so that saving a
// new default profile keeps the other values (and never persists values
// that only came from env vars or flags).
func loadOrCreateGlobal() *config.Config {
	return config.LoadGlobalConfig()
}

func isTerminal(r io.Reader) bool {
	f, ok := r.(*os.File)
	return ok && term.IsTerminal(int(f.Fd()))
}
