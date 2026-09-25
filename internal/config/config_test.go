package config

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// isolate points the config dir at a temp directory and clears every env
// variable that feeds Load, so tests never see the developer's real setup.
func isolate(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("OPENALGO_CONFIG_DIR", dir)
	for _, k := range []string{"OPENALGO_API_KEY", "OPENALGO_HOST", "OPENALGO_WS_URL", "OPENALGO_PROFILE", "OPENALGO_OUTPUT", "OPENALGO_QUIET", "OPENALGO_VERBOSE", "OPENALGO_DEBUG", "OPENALGO_TRACE"} {
		t.Setenv(k, "")
	}
	return dir
}

func TestNormalizeHost(t *testing.T) {
	cases := []struct {
		input string
		want  string
	}{
		{"", "http://127.0.0.1:5000"},
		{"   ", "http://127.0.0.1:5000"},
		{"http://127.0.0.1:5000", "http://127.0.0.1:5000"},
		{"http://127.0.0.1:5000/", "http://127.0.0.1:5000"},
		{"http://127.0.0.1:5000/api/v1", "http://127.0.0.1:5000"},
		{"http://127.0.0.1:5000/api/v1/", "http://127.0.0.1:5000"},
		{"localhost:5000", "http://localhost:5000"},
		{"https://algo.example.com", "https://algo.example.com"},
		{" https://algo.example.com// ", "https://algo.example.com"},
	}
	for _, tc := range cases {
		if got := NormalizeHost(tc.input); got != tc.want {
			t.Errorf("NormalizeHost(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

func TestBaseURL(t *testing.T) {
	if got := BaseURL("http://127.0.0.1:5000/"); got != "http://127.0.0.1:5000/api/v1" {
		t.Errorf("BaseURL = %q", got)
	}
	if got := BaseURL("https://algo.example.com/api/v1"); got != "https://algo.example.com/api/v1" {
		t.Errorf("BaseURL did not avoid a doubled prefix: %q", got)
	}
}

func TestWebSocketURL(t *testing.T) {
	cases := []struct {
		host string
		want string
	}{
		{"", "ws://127.0.0.1:8765"},
		{"http://127.0.0.1:5000", "ws://127.0.0.1:8765"},
		{"http://192.168.1.10:5000/api/v1", "ws://192.168.1.10:8765"},
		{"localhost", "ws://localhost:8765"},
		{"http://[::1]:5000", "ws://[::1]:8765"},
		{"http://127.0.0.1:5001", "ws://127.0.0.1:8766"},
		{"http://127.0.0.1:5002/", "ws://127.0.0.1:8767"},
		{"http://10.0.0.5:8080", "ws://10.0.0.5:8765"},
		{"https://algo.example.com", "wss://algo.example.com/ws"},
		{"https://algo.example.com:8443/", "wss://algo.example.com:8443/ws"},
	}
	for _, tc := range cases {
		if got := WebSocketURL(tc.host); got != tc.want {
			t.Errorf("WebSocketURL(%q) = %q, want %q", tc.host, got, tc.want)
		}
	}
}

func TestResolve(t *testing.T) {
	cases := []struct {
		values []string
		want   string
	}{
		{[]string{"", "", ""}, ""},
		{[]string{"", "b", "c"}, "b"},
		{[]string{"a", "b", "c"}, "a"},
		{[]string{"", "", "c"}, "c"},
	}
	for _, tc := range cases {
		if got := resolve(tc.values...); got != tc.want {
			t.Errorf("resolve(%v) = %q, want %q", tc.values, got, tc.want)
		}
	}
}

func TestLoad_NoCredentials(t *testing.T) {
	isolate(t)
	r, err := Load("", "")
	if err != nil {
		t.Fatal(err)
	}
	if r.HasCredentials() {
		t.Error("expected no credentials")
	}
	if r.Validate() == nil {
		t.Error("Validate should fail without credentials")
	}
	if r.ProfileName != DefaultProfileName {
		t.Errorf("ProfileName = %q, want %q", r.ProfileName, DefaultProfileName)
	}
	if r.BaseURL != "http://127.0.0.1:5000/api/v1" {
		t.Errorf("BaseURL = %q", r.BaseURL)
	}
	if r.WSURL != "ws://127.0.0.1:8765" {
		t.Errorf("WSURL = %q", r.WSURL)
	}
	if r.Output != "json" {
		t.Errorf("Output = %q, want json", r.Output)
	}
}

func TestLoad_EnvBundle(t *testing.T) {
	isolate(t)
	t.Setenv("OPENALGO_API_KEY", "env-key")
	t.Setenv("OPENALGO_HOST", "https://algo.example.com/")

	r, _ := Load("", "")
	if r.Source != SourceEnv || r.APIKey != "env-key" {
		t.Fatalf("got source=%q key=%q", r.Source, r.APIKey)
	}
	if r.BaseURL != "https://algo.example.com/api/v1" {
		t.Errorf("BaseURL = %q", r.BaseURL)
	}
	if r.WSURL != "wss://algo.example.com/ws" {
		t.Errorf("WSURL = %q", r.WSURL)
	}
}

func TestLoad_EnvWSOverride(t *testing.T) {
	isolate(t)
	t.Setenv("OPENALGO_API_KEY", "env-key")
	t.Setenv("OPENALGO_WS_URL", "ws://10.0.0.5:9000")

	r, _ := Load("", "")
	if r.WSURL != "ws://10.0.0.5:9000" {
		t.Errorf("WSURL = %q, want env override", r.WSURL)
	}
}

func TestLoad_ProfileBundle(t *testing.T) {
	isolate(t)
	if err := SaveProfile("default", &Profile{Host: "http://10.0.0.2:5000", APIKey: "profile-key"}); err != nil {
		t.Fatal(err)
	}

	r, _ := Load("", "")
	if r.Source != SourceProfile || r.APIKey != "profile-key" {
		t.Fatalf("got source=%q key=%q", r.Source, r.APIKey)
	}
	if r.BaseURL != "http://10.0.0.2:5000/api/v1" {
		t.Errorf("BaseURL = %q", r.BaseURL)
	}
	if r.WSURL != "ws://10.0.0.2:8765" {
		t.Errorf("WSURL = %q", r.WSURL)
	}
}

func TestLoad_ProfileWSURL(t *testing.T) {
	isolate(t)
	_ = SaveProfile("default", &Profile{Host: "http://10.0.0.2:5000", APIKey: "k", WSURL: "ws://10.0.0.2:9999"})

	r, _ := Load("", "")
	if r.WSURL != "ws://10.0.0.2:9999" {
		t.Errorf("WSURL = %q, want profile ws_url", r.WSURL)
	}
}

// TestLoad_NoMixing verifies the atomic bundle: env host/ws variables must
// never redirect a stored profile key to another server.
func TestLoad_NoMixing(t *testing.T) {
	isolate(t)
	_ = SaveProfile("default", &Profile{Host: "http://10.0.0.2:5000", APIKey: "profile-key"})
	t.Setenv("OPENALGO_HOST", "https://evil.example.com")
	t.Setenv("OPENALGO_WS_URL", "wss://evil.example.com/ws")

	r, _ := Load("", "")
	if r.APIKey != "profile-key" {
		t.Fatalf("APIKey = %q", r.APIKey)
	}
	if r.Host != "http://10.0.0.2:5000" {
		t.Errorf("Host = %q; OPENALGO_HOST must not apply to a profile key", r.Host)
	}
	if r.WSURL != "ws://10.0.0.2:8765" {
		t.Errorf("WSURL = %q; OPENALGO_WS_URL must not apply to a profile key", r.WSURL)
	}
}

// TestLoad_EnvKeyIgnoresProfileHost verifies the reverse: an env key goes to
// the env host (or the default), never to the profile's host.
func TestLoad_EnvKeyIgnoresProfileHost(t *testing.T) {
	isolate(t)
	_ = SaveProfile("default", &Profile{Host: "http://10.0.0.2:5000", APIKey: "profile-key"})
	t.Setenv("OPENALGO_API_KEY", "env-key")

	r, _ := Load("", "")
	if r.APIKey != "env-key" || r.Host != DefaultHost {
		t.Errorf("got key=%q host=%q, want env-key on %s", r.APIKey, r.Host, DefaultHost)
	}
}

func TestLoad_ProfileSelection(t *testing.T) {
	isolate(t)
	_ = SaveProfile("default", &Profile{APIKey: "default-key"})
	_ = SaveProfile("prod", &Profile{APIKey: "prod-key"})
	_ = SaveProfile("staging", &Profile{APIKey: "staging-key"})

	if r, _ := Load("prod", ""); r.APIKey != "prod-key" || r.ProfileName != "prod" {
		t.Errorf("flag: got %q/%q", r.ProfileName, r.APIKey)
	}

	t.Setenv("OPENALGO_PROFILE", "staging")
	if r, _ := Load("", ""); r.APIKey != "staging-key" {
		t.Errorf("env: got %q", r.APIKey)
	}
	if r, _ := Load("prod", ""); r.APIKey != "prod-key" {
		t.Errorf("flag should beat env: got %q", r.APIKey)
	}

	t.Setenv("OPENALGO_PROFILE", "")
	_ = SaveGlobalConfig(&Config{DefaultProfile: "prod"})
	if r, _ := Load("", ""); r.APIKey != "prod-key" {
		t.Errorf("global default: got %q", r.APIKey)
	}
}

func TestLoad_OutputPrecedence(t *testing.T) {
	isolate(t)
	if r, _ := Load("", "csv"); r.Output != "csv" {
		t.Errorf("flag: Output = %q", r.Output)
	}
	t.Setenv("OPENALGO_OUTPUT", "csv")
	if r, _ := Load("", ""); r.Output != "csv" {
		t.Errorf("env: Output = %q", r.Output)
	}
}

func TestSaveProfile_Permissions(t *testing.T) {
	dir := isolate(t)
	if err := SaveProfile("default", &Profile{APIKey: "k"}); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(filepath.Join(dir, "profiles", "default.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	// Windows has no Unix permission bits; the file inherits the user
	// profile's ACL instead.
	if perm := info.Mode().Perm(); runtime.GOOS != "windows" && perm != 0o600 {
		t.Errorf("profile permissions = %o, want 600", perm)
	}
}

func TestListAndDeleteProfiles(t *testing.T) {
	isolate(t)
	if names, err := ListProfiles(); err != nil || len(names) != 0 {
		t.Fatalf("expected no profiles, got %v (%v)", names, err)
	}
	_ = SaveProfile("a", &Profile{APIKey: "1"})
	_ = SaveProfile("b", &Profile{APIKey: "2"})
	names, _ := ListProfiles()
	if len(names) != 2 || names[0] != "a" || names[1] != "b" {
		t.Errorf("ListProfiles = %v", names)
	}
	if err := DeleteProfile("a"); err != nil {
		t.Fatal(err)
	}
	names, _ = ListProfiles()
	if len(names) != 1 || names[0] != "b" {
		t.Errorf("after delete = %v", names)
	}
}

func TestResolved_HasCredentials(t *testing.T) {
	cases := []struct {
		source Source
		want   bool
	}{
		{SourceEnv, true},
		{SourceProfile, true},
		{SourceNone, false},
	}
	for _, tc := range cases {
		r := &Resolved{Source: tc.source}
		if got := r.HasCredentials(); got != tc.want {
			t.Errorf("Source=%q HasCredentials()=%v, want %v", tc.source, got, tc.want)
		}
	}
}

func TestValidateProfileName(t *testing.T) {
	for _, ok := range []string{"default", "live", "office_2", "a.b-c"} {
		if err := ValidateProfileName(ok); err != nil {
			t.Errorf("ValidateProfileName(%q) = %v, want nil", ok, err)
		}
	}
	for _, bad := range []string{"", "../evil", "a/b", `a\b`, ".hidden", "-x", "has space"} {
		if err := ValidateProfileName(bad); err == nil {
			t.Errorf("ValidateProfileName(%q) = nil, want error", bad)
		}
	}
}

func TestProfileNameCannotEscapeDir(t *testing.T) {
	dir := isolate(t)
	if err := SaveProfile("../evil", &Profile{APIKey: "k"}); err == nil {
		t.Fatal("SaveProfile accepted ../evil")
	}
	if _, err := os.Stat(filepath.Join(dir, "evil.yaml")); !os.IsNotExist(err) {
		t.Error("a file was written outside profiles/")
	}
	if err := DeleteProfile("../config"); err == nil {
		t.Error("DeleteProfile accepted a path")
	}
	if _, err := Load("../evil", ""); err == nil {
		t.Error("Load accepted a path as the profile name")
	}
}

func TestLoad_ExplicitMissingProfile(t *testing.T) {
	isolate(t)
	if err := SaveProfile("live", &Profile{APIKey: "k"}); err != nil {
		t.Fatal(err)
	}
	_, err := Load("nope", "")
	if err == nil || !strings.Contains(err.Error(), `profile "nope" not found`) || !strings.Contains(err.Error(), "live") {
		t.Fatalf("Load(nope) = %v, want not-found error listing live", err)
	}
	t.Setenv("OPENALGO_PROFILE", "nope")
	if _, err := Load("", ""); err == nil {
		t.Error("OPENALGO_PROFILE naming a missing profile should fail")
	}
	// With env credentials the profile is not used, so it need not exist.
	t.Setenv("OPENALGO_API_KEY", "envkey")
	if _, err := Load("nope", ""); err != nil {
		t.Errorf("env credentials: %v", err)
	}
}

func TestValidateHost(t *testing.T) {
	for _, ok := range []string{"http://127.0.0.1:5000", "https://algo.example.com"} {
		if err := ValidateHost(ok); err != nil {
			t.Errorf("ValidateHost(%q) = %v", ok, err)
		}
	}
	for _, bad := range []string{"http://not a url", "ftp://host", "http://"} {
		if err := ValidateHost(bad); err == nil {
			t.Errorf("ValidateHost(%q) = nil, want error", bad)
		}
	}
}
