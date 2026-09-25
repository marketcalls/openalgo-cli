package cmd

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/marketcalls/openalgo-cli/internal/client"

	"github.com/fatih/color"
	"github.com/marketcalls/openalgo-cli/internal/config"
	"github.com/spf13/cobra"
)

// isolateConfig points the config dir at a temp dir and clears the env
// variables that feed config.Load.
func isolateConfig(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("OPENALGO_CONFIG_DIR", dir)
	for _, k := range []string{"OPENALGO_API_KEY", "OPENALGO_HOST", "OPENALGO_WS_URL", "OPENALGO_PROFILE", "OPENALGO_OUTPUT", "OPENALGO_QUIET", "OPENALGO_VERBOSE", "OPENALGO_DEBUG", "OPENALGO_TRACE"} {
		t.Setenv(k, "")
	}
	return dir
}

// captureColorOutput redirects color.Output (where color.Green/Yellow
// write) to a buffer and disables color codes for the duration of the test.
func captureColorOutput(t *testing.T) *bytes.Buffer {
	t.Helper()
	buf := new(bytes.Buffer)
	oldOut, oldNoColor := color.Output, color.NoColor
	color.Output, color.NoColor = buf, true
	t.Cleanup(func() { color.Output, color.NoColor = oldOut, oldNoColor })
	return buf
}

// silenceStderr discards the prompts and notes login writes to os.Stderr.
func silenceStderr(t *testing.T) {
	t.Helper()
	devnull, err := os.Open(os.DevNull)
	if err != nil {
		t.Fatal(err)
	}
	old := os.Stderr
	os.Stderr = devnull
	t.Cleanup(func() { os.Stderr = old; _ = devnull.Close() })
}

// runLogin invokes loginWithAPIKey on a throwaway cobra.Command. We don't use
// Root().Execute() because cobra auto-registers completion/help subcommands
// on first Execute(), polluting the shared rootCmd tree.
func runLogin(t *testing.T, stdin string, flags map[string]string) error {
	t.Helper()
	old := loginInput
	loginInput = strings.NewReader(stdin)
	t.Cleanup(func() { loginInput = old })

	cmd := &cobra.Command{Use: "login"}
	cmd.Flags().String("host", "", "")
	cmd.Flags().String("key", "", "")
	cmd.Flags().String("name", "", "")
	cmd.Flags().String("ws-url", "", "")
	cmd.Flags().Bool("no-validate", false, "")
	for k, v := range flags {
		if err := cmd.Flags().Set(k, v); err != nil {
			t.Fatal(err)
		}
	}
	return loginWithAPIKey(cmd)
}

// pingServer mimics POST /api/v1/ping, accepting only goodKey.
func pingServer(t *testing.T, goodKey string) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" || r.URL.Path != "/api/v1/ping" {
			w.WriteHeader(404)
			return
		}
		var body map[string]string
		_ = json.NewDecoder(r.Body).Decode(&body)
		if body["apikey"] != goodKey {
			w.WriteHeader(403)
			_, _ = w.Write([]byte(`{"status":"error","message":"Invalid openalgo apikey"}`))
			return
		}
		_, _ = w.Write([]byte(`{"status":"success","data":{"broker":"zerodha","message":"pong"}}`))
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestLogin_ValidatesAndSaves(t *testing.T) {
	dir := isolateConfig(t)
	silenceStderr(t)
	out := captureColorOutput(t)
	srv := pingServer(t, "good-key")

	if err := runLogin(t, "good-key\n", map[string]string{"host": srv.URL + "/api/v1/"}); err != nil {
		t.Fatalf("login failed: %v", err)
	}
	p := config.LoadProfileByName("default")
	if p.APIKey != "good-key" || p.Host != srv.URL {
		t.Errorf("saved profile = %+v, want key and normalized host %s", p, srv.URL)
	}
	if !strings.Contains(out.String(), "broker: zerodha") {
		t.Errorf("output = %q, want broker name", out.String())
	}
	info, err := os.Stat(filepath.Join(dir, "profiles", "default.yaml"))
	if err != nil || (runtime.GOOS != "windows" && info.Mode().Perm() != 0o600) {
		t.Errorf("profile file mode = %v (%v), want 0600", info.Mode().Perm(), err)
	}
	if r, _ := config.Load("", ""); r.ProfileName != "default" || r.APIKey != "good-key" {
		t.Errorf("login did not become the active profile: %+v", r)
	}
}

func TestLogin_RejectsBadKey(t *testing.T) {
	isolateConfig(t)
	silenceStderr(t)
	captureColorOutput(t)
	srv := pingServer(t, "good-key")

	err := runLogin(t, "", map[string]string{"host": srv.URL, "key": "bad-key"})
	if err == nil || !strings.Contains(err.Error(), "invalid API key") {
		t.Fatalf("err = %v, want invalid API key", err)
	}
	if strings.Contains(err.Error(), "bad-key") {
		t.Error("error message leaks the key")
	}
	if p := config.LoadProfileByName("default"); p.APIKey != "" {
		t.Error("a rejected key must not be saved")
	}
}

func TestLogin_RejectsNonOpenAlgoHost(t *testing.T) {
	isolateConfig(t)
	silenceStderr(t)
	captureColorOutput(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("<html>hello</html>"))
	}))
	t.Cleanup(srv.Close)

	if err := runLogin(t, "", map[string]string{"host": srv.URL, "key": "k"}); err == nil {
		t.Fatal("expected an error for a host that is not OpenAlgo")
	}
}

func TestLogin_PipedKeyNoValidate(t *testing.T) {
	isolateConfig(t)
	silenceStderr(t)
	captureColorOutput(t)

	// Non-terminal stdin: no host prompt, the first line is the key.
	if err := runLogin(t, "piped-key\n", map[string]string{"name": "office", "no-validate": "true"}); err != nil {
		t.Fatal(err)
	}
	p := config.LoadProfileByName("office")
	if p.APIKey != "piped-key" || p.Host != config.DefaultHost {
		t.Errorf("profile = %+v", p)
	}
}

func TestLogin_RequiresKey(t *testing.T) {
	isolateConfig(t)
	silenceStderr(t)
	captureColorOutput(t)
	if err := runLogin(t, "\n", map[string]string{"no-validate": "true"}); err == nil {
		t.Error("expected error when no key is given")
	}
}

func TestLogin_KeepsProfileWSURL(t *testing.T) {
	isolateConfig(t)
	silenceStderr(t)
	captureColorOutput(t)
	_ = config.SaveProfile("default", &config.Profile{APIKey: "old", WSURL: "ws://10.0.0.9:9000"})

	if err := runLogin(t, "new\n", map[string]string{"no-validate": "true"}); err != nil {
		t.Fatal(err)
	}
	if p := config.LoadProfileByName("default"); p.WSURL != "ws://10.0.0.9:9000" || p.APIKey != "new" {
		t.Errorf("profile = %+v, want ws_url kept", p)
	}
}

func TestLogin_SetsWSURL(t *testing.T) {
	isolateConfig(t)
	silenceStderr(t)
	captureColorOutput(t)
	if err := runLogin(t, "k\n", map[string]string{"no-validate": "true", "ws-url": "wss://ws.example.com"}); err != nil {
		t.Fatal(err)
	}
	if p := config.LoadProfileByName("default"); p.WSURL != "wss://ws.example.com" {
		t.Errorf("profile = %+v, want ws_url set", p)
	}
}

// TestLogin_WarnsWhenEnvShadows verifies that logging in while
// OPENALGO_API_KEY is set triggers the shadowing warning - users must know
// their fresh profile is about to be overridden by env.
func TestLogin_WarnsWhenEnvShadows(t *testing.T) {
	isolateConfig(t)
	silenceStderr(t)
	out := captureColorOutput(t)
	t.Setenv("OPENALGO_API_KEY", "env-key")

	if err := runLogin(t, "new-key\n", map[string]string{"name": "prod", "no-validate": "true"}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "OPENALGO_API_KEY is set in your environment") || !strings.Contains(out.String(), `"prod"`) {
		t.Errorf("expected shadow warning naming prod, got: %q", out.String())
	}
}

func TestLogin_NoWarnWhenEnvUnset(t *testing.T) {
	isolateConfig(t)
	silenceStderr(t)
	out := captureColorOutput(t)

	if err := runLogin(t, "new-key\n", map[string]string{"no-validate": "true"}); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out.String(), "OPENALGO_API_KEY is set") {
		t.Errorf("unexpected shadow warning: %q", out.String())
	}
}

func TestEnvShadowsProfile(t *testing.T) {
	isolateConfig(t)
	_ = config.SaveProfile("default", &config.Profile{APIKey: "profile-key"})

	if envShadowsProfile("default") {
		t.Error("no shadow expected without OPENALGO_API_KEY")
	}
	t.Setenv("OPENALGO_API_KEY", "env-key")
	if !envShadowsProfile("default") {
		t.Error("expected shadowing when env key set and profile has a key")
	}
	if envShadowsProfile("missing") {
		t.Error("no shadow expected when the profile file is absent")
	}
}

func TestLoadOrCreateGlobal_DoesNotPersistEnv(t *testing.T) {
	isolateConfig(t)
	t.Setenv("OPENALGO_OUTPUT", "csv")
	if g := loadOrCreateGlobal(); g.Output != "" {
		t.Errorf("Output = %q; env-only settings must not be written to config.yaml", g.Output)
	}
}

func TestCredentialSourceDescription(t *testing.T) {
	if got := credentialSourceDescription(&config.Resolved{Source: config.SourceEnv}); !strings.Contains(got, "OPENALGO_API_KEY") {
		t.Errorf("env = %q", got)
	}
	if got := credentialSourceDescription(&config.Resolved{Source: config.SourceProfile, ProfileName: "office"}); !strings.Contains(got, `"office"`) {
		t.Errorf("profile = %q", got)
	}
}

func TestLogin_BadKeyIsAuthError(t *testing.T) {
	isolateConfig(t)
	silenceStderr(t)
	captureColorOutput(t)
	srv := pingServer(t, "good-key")

	err := runLogin(t, "", map[string]string{"host": srv.URL, "key": "bad-key"})
	var apiErr *client.APIError
	if !errors.As(err, &apiErr) || apiErr.ExitCode() != client.ExitAuthError {
		t.Fatalf("err = %#v, want an APIError with exit code 2", err)
	}
}

func TestLogin_RejectsBadNameAndHost(t *testing.T) {
	dir := isolateConfig(t)
	silenceStderr(t)
	captureColorOutput(t)

	if err := runLogin(t, "", map[string]string{"name": "../evil", "key": "k", "no-validate": "true"}); err == nil {
		t.Error("login accepted ../evil as a profile name")
	}
	if _, err := os.Stat(filepath.Join(dir, "evil.yaml")); !os.IsNotExist(err) {
		t.Error("login wrote a file outside profiles/")
	}
	if err := runLogin(t, "", map[string]string{"host": "not a url", "key": "k", "no-validate": "true"}); err == nil {
		t.Error("login accepted an invalid host")
	}
}

func TestLogout_ActiveProfileMovesDefault(t *testing.T) {
	isolateConfig(t)
	for _, name := range []string{"live", "office"} {
		if err := config.SaveProfile(name, &config.Profile{APIKey: "k"}); err != nil {
			t.Fatal(err)
		}
	}
	if err := config.SaveGlobalConfig(&config.Config{DefaultProfile: "live"}); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	profileLogoutCmd.SetOut(&out)
	t.Cleanup(func() { profileLogoutCmd.SetOut(nil) })
	if err := profileLogoutCmd.RunE(profileLogoutCmd, []string{"live"}); err != nil {
		t.Fatal(err)
	}
	if got := config.LoadGlobalConfig().DefaultProfile; got != "office" {
		t.Errorf("default_profile = %q, want office", got)
	}
	if !strings.Contains(out.String(), "Active profile is now office") {
		t.Errorf("output = %q", out.String())
	}
	if err := profileLogoutCmd.RunE(profileLogoutCmd, []string{"office"}); err != nil {
		t.Fatal(err)
	}
	if got := config.LoadGlobalConfig().DefaultProfile; got != "" {
		t.Errorf("default_profile = %q, want empty after the last profile is removed", got)
	}
}
