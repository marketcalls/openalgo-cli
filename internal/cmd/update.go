package cmd

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/marketcalls/openalgo-cli/internal/cmdutil"
	"github.com/marketcalls/openalgo-cli/internal/useragent"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

const (
	repoOwner = "marketcalls"
	repoName  = "openalgo-cli"
)

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Check for updates and (optionally) install them",
	Long: `Check for a newer version of the OpenAlgo CLI.

Queries GitHub for the latest release, detects your install method
(Homebrew, go install, or the install script), and prompts to run the upgrade. Use --check
to print structured JSON without prompting.`,
	Example: `  openalgo update
  openalgo update --yes
  openalgo update --check`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		checkOnly := cmdutil.Bool(cmd, "check")
		assumeYes := cmdutil.Bool(cmd, "yes")

		latest, err := getLatestVersion(10 * time.Second)
		if errors.Is(err, errNoRelease) {
			// The repository has no published release yet: nothing to
			// update to, which is not a failure.
			if checkOnly {
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetIndent("", "  ")
				return enc.Encode(map[string]any{
					"current":          strings.TrimPrefix(version, "v"),
					"latest":           nil,
					"update_available": false,
					"note":             "no published release found",
				})
			}
			fmt.Printf("No published release found for %s/%s; current version %s.\n", repoOwner, repoName, version)
			return nil
		}
		if err != nil {
			return fmt.Errorf("checking for updates: %w", err)
		}

		method := detectInstallMethod()
		current := version
		upToDate := !versionNewer(latest, current)
		upgradeCmd := upgradeCommand(method)

		if checkOnly {
			m := map[string]any{
				"current":          strings.TrimPrefix(current, "v"),
				"latest":           strings.TrimPrefix(latest, "v"),
				"update_available": !upToDate,
				"install_method":   method,
				"update_command":   upgradeCmd,
			}
			enc := json.NewEncoder(cmd.OutOrStdout())
			enc.SetIndent("", "  ")
			return enc.Encode(m)
		}

		if upToDate {
			color.Green("Already up to date (%s)", current)
			return nil
		}

		fmt.Printf("Current version: %s\n", current)
		fmt.Printf("Latest version:  %s\n", latest)
		fmt.Println()

		if !assumeYes && !term.IsTerminal(int(os.Stdin.Fd())) {
			fmt.Println("To update, run:")
			fmt.Println("  " + upgradeCmd)
			return nil
		}

		if !assumeYes {
			fmt.Printf("Upgrade %s -> %s using `%s`? [y/N] ", current, latest, upgradeCmd)
			reader := bufio.NewReader(os.Stdin)
			ans, _ := reader.ReadString('\n')
			ans = strings.ToLower(strings.TrimSpace(ans))
			if ans != "y" && ans != "yes" {
				fmt.Println("Canceled. To update later, run:")
				fmt.Println("  " + upgradeCmd)
				return nil
			}
		}

		return runUpgrade(upgradeCmd)
	},
}

// runUpgrade shells out to the install-method-specific upgrade command and
// streams its output through to the user's terminal. We use sh -c (or
// PowerShell on Windows) so users see the same command they'd run by hand,
// without us having to parse its arguments.
func runUpgrade(upgradeCmd string) error {
	fmt.Fprintln(os.Stderr, "Running:", upgradeCmd)
	c := exec.Command("sh", "-c", upgradeCmd)
	if runtime.GOOS == goosWindows {
		c = exec.Command("powershell", "-NoProfile", "-Command", upgradeCmd)
		// Started from PowerShell 7, Windows PowerShell inherits a
		// PSModulePath that points at PowerShell 7's modules and cannot load
		// its own script modules. Dropping the variable lets it rebuild its
		// default.
		c.Env = withoutEnv(os.Environ(), "PSModulePath")
	}
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	c.Stdin = os.Stdin
	if err := c.Run(); err != nil {
		return fmt.Errorf("upgrade command failed: %w", err)
	}
	return nil
}

func init() {
	updateCmd.Flags().Bool("check", false, "Print update status as JSON without prompting")
	updateCmd.Flags().Bool("yes", false, "Skip the confirmation prompt and run the upgrade immediately")
}

// printUpdateNoticeIfAvailable does a best-effort GitHub check and prints a
// short notice when a newer release exists. It mirrors the doctor command's
// "Update:" section but stays silent on errors and when already up to date,
// since the bare `openalgo` command is mostly help output and any noise here
// would be surfaced on every invocation.
func printUpdateNoticeIfAvailable(w io.Writer, timeout time.Duration) {
	latest, err := getLatestVersion(timeout)
	if err != nil || !versionNewer(latest, version) {
		return
	}
	method := detectInstallMethod()
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Update:")
	fmt.Fprintf(w, "  - update available: %s -> %s, run `%s` (or `openalgo update`)\n",
		version, latest, upgradeCommand(method))
}

// errNoRelease means GitHub has no published release for the repository
// (the releases/latest endpoint answers 404).
var errNoRelease = errors.New("no published release found")

type ghRelease struct {
	TagName string `json:"tag_name"`
}

func getLatestVersion(timeout time.Duration) (string, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/latest", repoOwner, repoName)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", useragent.Build(version))
	resp, err := (&http.Client{Timeout: timeout}).Do(req)
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusNotFound {
		return "", errNoRelease
	}
	if resp.StatusCode == http.StatusForbidden || resp.StatusCode == http.StatusTooManyRequests {
		// Anonymous API calls are limited to 60/hour per IP, which shared
		// and NAT'd networks exhaust. The releases/latest page redirect is
		// not rate limited the same way.
		return latestFromRedirect(timeout)
	}
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("GitHub API returned %d", resp.StatusCode)
	}

	var release ghRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return "", err
	}

	return release.TagName, nil
}

// releasesLatestURL is the page GitHub redirects to the newest release tag.
var releasesLatestURL = fmt.Sprintf("https://github.com/%s/%s/releases/latest", repoOwner, repoName)

// latestFromRedirect reads the newest tag from the Location header of the
// releases/latest redirect (".../releases/tag/v1.2.3").
func latestFromRedirect(timeout time.Duration) (string, error) {
	req, err := http.NewRequest("HEAD", releasesLatestURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", useragent.Build(version))
	c := &http.Client{
		Timeout:       timeout,
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}
	resp, err := c.Do(req)
	if err != nil {
		return "", err
	}
	_ = resp.Body.Close()
	loc := resp.Header.Get("Location")
	if i := strings.LastIndex(loc, "/releases/tag/"); i >= 0 {
		return loc[i+len("/releases/tag/"):], nil
	}
	if resp.StatusCode == http.StatusNotFound || strings.HasSuffix(loc, "/releases") {
		return "", errNoRelease
	}
	return "", fmt.Errorf("could not determine the latest release (HTTP %d)", resp.StatusCode)
}

// withoutEnv returns env without the named variable (case-insensitive, as
// on Windows).
func withoutEnv(env []string, name string) []string {
	out := make([]string, 0, len(env))
	for _, kv := range env {
		if k, _, _ := strings.Cut(kv, "="); strings.EqualFold(k, name) {
			continue
		}
		out = append(out, kv)
	}
	return out
}
