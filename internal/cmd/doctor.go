package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"runtime"
	"time"

	"github.com/fatih/color"
	"github.com/marketcalls/openalgo-cli/internal/client"
	"github.com/marketcalls/openalgo-cli/internal/config"
	"github.com/spf13/cobra"
)

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Check CLI configuration and connectivity",
	Long: `Run diagnostic checks on your OpenAlgo CLI setup: config files, credentials,
server connectivity (ping), analyzer (sandbox) mode, the WebSocket URL, and
available updates.`,
	Example: `  openalgo doctor
  openalgo doctor --profile office`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runDoctor(cmd.OutOrStdout(), profileFlag, true)
	},
}

// runDoctor prints the diagnostic report. checkUpdate is false in tests so
// they never reach GitHub.
func runDoctor(w io.Writer, profile string, checkUpdate bool) error {
	allOK := true
	usingEnvCredentials := os.Getenv("OPENALGO_API_KEY") != ""

	fmt.Fprintf(w, "OpenAlgo CLI %s\n", version)
	fmt.Fprintf(w, "  Go:       %s\n", runtime.Version())
	fmt.Fprintf(w, "  OS/Arch:  %s/%s\n\n", runtime.GOOS, runtime.GOARCH)

	configDir := config.Dir()
	fmt.Fprintf(w, "Config:     %s\n", configDir)
	if _, err := os.Stat(configDir); os.IsNotExist(err) {
		if usingEnvCredentials {
			printCheck(w, true, "config directory does not exist (ok when using env vars)")
		} else {
			allOK = printCheck(w, false, "config directory does not exist")
		}
	} else {
		printCheck(w, true, "config directory exists")
	}

	profiles, _ := config.ListProfiles()
	if len(profiles) == 0 {
		if usingEnvCredentials {
			printCheck(w, true, "no saved profiles configured (using env var credentials)")
		} else {
			allOK = printCheck(w, false, "no profiles configured - run `openalgo profile login`")
		}
	} else {
		printCheck(w, true, fmt.Sprintf("%d profile(s): %s", len(profiles), joinMax(profiles, 5)))
	}

	resolved, err := config.Load(profile, "")
	if err != nil {
		allOK = printCheck(w, false, "failed to load config: "+err.Error())
		return doctorResult(w, allOK)
	}
	if usingEnvCredentials || config.ProfileExists(resolved.ProfileName) {
		printCheck(w, true, "active profile: "+resolved.ProfileName)
	} else {
		allOK = printCheck(w, false, fmt.Sprintf("active profile %q does not exist - run `openalgo profile switch <name>` or `openalgo profile login`", resolved.ProfileName))
	}

	if !resolved.HasCredentials() {
		allOK = printCheck(w, false, "no API key - run `openalgo profile login` or set OPENALGO_API_KEY")
		return doctorResult(w, allOK)
	}
	printCheck(w, true, credentialSourceDescription(resolved))
	warnEnvShadowsProfile(resolved.ProfileName, "  ")

	fmt.Fprintf(w, "\nConnectivity:\n")
	fmt.Fprintf(w, "  Host:      %s\n", resolved.Host)
	fmt.Fprintf(w, "  REST:      %s\n", resolved.BaseURL)
	fmt.Fprintf(w, "  WebSocket: %s\n", resolved.WSURL)
	c := client.New(resolved)
	c.SetTimeout(10 * time.Second)

	data, err := c.Post("/ping", nil)
	if err != nil {
		allOK = printCheck(w, false, "ping: "+err.Error())
		if apiErr, ok := err.(*client.APIError); ok && apiErr.Hint() != "" {
			fmt.Fprintf(w, "         %s\n", apiErr.Hint())
		}
	} else {
		var pong struct {
			Data struct {
				Broker string `json:"broker"`
			} `json:"data"`
		}
		_ = json.Unmarshal(data, &pong)
		msg := "ping: connected"
		if pong.Data.Broker != "" {
			msg += " (broker: " + pong.Data.Broker + ")"
		}
		printCheck(w, true, msg)

		if data, err := c.Post("/analyzer", nil); err != nil {
			allOK = printCheck(w, false, "analyzer status: "+err.Error())
		} else {
			printCheck(w, true, analyzerDescription(data))
		}
	}

	if checkUpdate {
		fmt.Fprintf(w, "\nUpdate:\n")
		latest, err := getLatestVersion(10 * time.Second)
		if errors.Is(err, errNoRelease) {
			fmt.Fprintf(w, "  - no published release found for %s/%s\n", repoOwner, repoName)
		} else if err != nil {
			fmt.Fprintf(w, "  - could not check for updates: %v\n", err)
		} else if !versionNewer(latest, version) {
			printCheck(w, true, fmt.Sprintf("up to date (%s)", version))
		} else {
			method := detectInstallMethod()
			fmt.Fprintf(w, "  - update available: %s -> %s, run `%s`\n",
				version, latest, upgradeCommand(method))
		}
	}

	return doctorResult(w, allOK)
}

// analyzerDescription renders the /analyzer response as a single line.
func analyzerDescription(data []byte) string {
	var st struct {
		Data struct {
			AnalyzeMode bool   `json:"analyze_mode"`
			Mode        string `json:"mode"`
			TotalLogs   int    `json:"total_logs"`
		} `json:"data"`
	}
	_ = json.Unmarshal(data, &st)
	if st.Data.AnalyzeMode {
		return fmt.Sprintf("analyzer (sandbox) mode: on (%d orders logged)", st.Data.TotalLogs)
	}
	return "analyzer (sandbox) mode: off (orders go to the live broker)"
}

func printCheck(w io.Writer, ok bool, msg string) bool {
	if ok {
		color.New(color.FgGreen).Fprintf(w, "  [ok]   ")
	} else {
		color.New(color.FgRed).Fprintf(w, "  [FAIL] ")
	}
	fmt.Fprintln(w, msg)
	return ok
}

func doctorResult(w io.Writer, allOK bool) error {
	fmt.Fprintln(w)
	if allOK {
		color.New(color.FgGreen).Fprintln(w, "All checks passed.")
		return nil
	}
	return fmt.Errorf("some checks failed")
}

func joinMax(items []string, max int) string {
	if len(items) <= max {
		return fmt.Sprintf("%v", items)
	}
	shown := items[:max]
	return fmt.Sprintf("%v (+%d more)", shown, len(items)-max)
}

func credentialSourceDescription(r *config.Resolved) string {
	switch r.Source {
	case config.SourceEnv:
		return "API key from env (OPENALGO_API_KEY, host from OPENALGO_HOST or default)"
	case config.SourceProfile:
		return fmt.Sprintf("API key from profile %q", r.ProfileName)
	default:
		return "credentials configured"
	}
}
