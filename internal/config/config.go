package config

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

const (
	// DefaultHost is the address a stock OpenAlgo install listens on.
	DefaultHost = "http://127.0.0.1:5000"
	// APIPrefix is appended to the host to build the REST base URL.
	APIPrefix = "/api/v1"
	// DefaultWSPort is the port the OpenAlgo WebSocket proxy listens on
	// when the REST server is reached directly over plain HTTP.
	DefaultWSPort = "8765"
	// defaultFlaskPort is the REST port of the first OpenAlgo instance.
	defaultFlaskPort = 5000

	DefaultProfileName = "default"
)

// Source identifies where the resolved credentials came from.
// Credentials resolve as an atomic bundle (API key + host) - never mixed
// across sources - because an OpenAlgo API key is only valid for the server
// that issued it. Sending a profile's key to an env-provided host (or the
// other way round) would leak the key to the wrong server.
type Source string

const (
	SourceNone    Source = ""
	SourceEnv     Source = "env"
	SourceProfile Source = "profile"
)

type Config struct {
	DefaultProfile string `yaml:"default_profile"`
	Output         string `yaml:"output"`
	Color          string `yaml:"color"`
}

type Profile struct {
	Host   string `yaml:"host"`
	APIKey string `yaml:"api_key"`
	// WSURL overrides the WebSocket URL derived from Host (optional).
	WSURL string `yaml:"ws_url,omitempty"`
}

type Resolved struct {
	APIKey      string
	Host        string
	BaseURL     string
	WSURL       string
	Output      string
	Color       string
	ProfileName string
	Source      Source
}

func Dir() string {
	if d := os.Getenv("OPENALGO_CONFIG_DIR"); d != "" {
		return d
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "openalgo")
}

// Load resolves credentials and URLs from env vars and the named profile.
//
// Credentials resolve as an atomic bundle - the first complete source wins,
// and field-level mixing across sources is not allowed. Order:
//  1. env OPENALGO_API_KEY (+ OPENALGO_HOST, default http://127.0.0.1:5000)
//  2. profile api_key (+ profile host, default http://127.0.0.1:5000)
//
// OPENALGO_HOST without OPENALGO_API_KEY is ignored so that a stray host
// variable can never redirect a stored profile key to another server.
// The WebSocket URL follows the same rule: OPENALGO_WS_URL belongs to the
// env bundle and the profile's ws_url to the profile bundle; when neither
// is set it is derived from the bundle's host (see WebSocketURL).
//
// A profile named explicitly (--profile or OPENALGO_PROFILE) must be a valid
// name and, unless OPENALGO_API_KEY supplies the credentials, must exist.
func Load(profileFlag, outputFlag string) (*Resolved, error) {
	cfg := loadGlobalConfig()
	explicit := resolve(profileFlag, os.Getenv("OPENALGO_PROFILE"))
	profileName := resolve(explicit, cfg.DefaultProfile, DefaultProfileName)
	if err := ValidateProfileName(profileName); err != nil {
		return nil, err
	}
	if explicit != "" && os.Getenv("OPENALGO_API_KEY") == "" && !ProfileExists(explicit) {
		return nil, ProfileNotFoundError(explicit)
	}
	profile := loadProfile(profileName)

	r := &Resolved{
		ProfileName: profileName,
		Output:      resolve(outputFlag, os.Getenv("OPENALGO_OUTPUT"), cfg.Output, "json"),
		Color:       resolve(cfg.Color, "auto"),
	}

	switch {
	case os.Getenv("OPENALGO_API_KEY") != "":
		r.APIKey = os.Getenv("OPENALGO_API_KEY")
		r.Host = NormalizeHost(os.Getenv("OPENALGO_HOST"))
		r.WSURL = resolve(strings.TrimSpace(os.Getenv("OPENALGO_WS_URL")), WebSocketURL(r.Host))
		r.Source = SourceEnv
	case profile.APIKey != "":
		r.APIKey = profile.APIKey
		r.Host = NormalizeHost(profile.Host)
		r.WSURL = resolve(strings.TrimSpace(profile.WSURL), WebSocketURL(r.Host))
		r.Source = SourceProfile
	default:
		r.Host = NormalizeHost(profile.Host)
		r.WSURL = resolve(strings.TrimSpace(profile.WSURL), WebSocketURL(r.Host))
		r.Source = SourceNone
	}
	r.BaseURL = BaseURL(r.Host)

	return r, nil
}

// profileNameRe allows letters, digits, '_', '.', and '-', starting with a
// letter, digit or '_' so a name can never be a path ("../x") or hidden file.
var profileNameRe = regexp.MustCompile(`^[A-Za-z0-9_][A-Za-z0-9_.-]{0,63}$`)

// ValidateProfileName rejects names that are not safe as a file name under
// profiles/, such as "../evil" or "a/b".
func ValidateProfileName(name string) error {
	if !profileNameRe.MatchString(name) {
		return fmt.Errorf("invalid profile name %q\nHint: use up to 64 letters, digits, '_', '.' or '-', not starting with '.' or '-'", name)
	}
	return nil
}

// ProfileExists reports whether a profile file exists for name.
func ProfileExists(name string) bool {
	if ValidateProfileName(name) != nil {
		return false
	}
	_, err := os.Stat(profilePath(name))
	return err == nil
}

// ProfileNotFoundError reports a missing profile and lists the ones that
// exist.
func ProfileNotFoundError(name string) error {
	available := "(none)"
	if names, _ := ListProfiles(); len(names) > 0 {
		available = strings.Join(names, ", ")
	}
	return fmt.Errorf("profile %q not found\nHint: available profiles: %s; run `openalgo profile login --name %s` to create it", name, available, name)
}

func profilePath(name string) string {
	return filepath.Join(Dir(), "profiles", name+".yaml")
}

// ValidateHost checks a normalized host is a usable http(s) URL.
func ValidateHost(host string) error {
	u, err := url.Parse(host)
	if err != nil || strings.ContainsAny(host, " \t") || u.Hostname() == "" || (u.Scheme != "http" && u.Scheme != "https") {
		return fmt.Errorf("invalid host %q\nHint: use an http or https URL such as %s", host, DefaultHost)
	}
	return nil
}

// NormalizeHost trims whitespace and trailing slashes, strips a trailing
// /api/v1 if the user pasted the full API URL, and defaults to localhost.
func NormalizeHost(host string) string {
	host = strings.TrimSpace(host)
	if host == "" {
		return DefaultHost
	}
	host = strings.TrimRight(host, "/")
	host = strings.TrimSuffix(host, APIPrefix)
	if !strings.Contains(host, "://") {
		host = "http://" + host
	}
	return strings.TrimRight(host, "/")
}

// BaseURL returns the REST base URL for a host.
func BaseURL(host string) string {
	return NormalizeHost(host) + APIPrefix
}

// WebSocketURL derives the streaming endpoint from a REST host.
//
// A plain-HTTP host is a direct install where the WebSocket proxy listens
// on its own port (http://127.0.0.1:5000 -> ws://127.0.0.1:8765). Multiple
// instances on one machine shift both ports by the same offset (instance n
// uses 5000+n-1 and 8765+n-1), so http://127.0.0.1:5001 -> ws://127.0.0.1:8766.
// An HTTPS host sits behind a reverse proxy that forwards /ws to the proxy
// (https://algo.example.com -> wss://algo.example.com/ws).
func WebSocketURL(host string) string {
	u, err := url.Parse(NormalizeHost(host))
	if err != nil || u.Hostname() == "" {
		return "ws://127.0.0.1:" + DefaultWSPort
	}
	if u.Scheme == "https" {
		return "wss://" + u.Host + "/ws"
	}
	h := u.Hostname()
	if strings.Contains(h, ":") {
		h = "[" + h + "]" // IPv6 literal
	}
	return "ws://" + h + ":" + wsPortFor(u.Port())
}

// maxInstances bounds the port offset treated as a multi-instance layout.
// Ports outside 5000..5000+maxInstances-1 are custom setups, which fall back
// to the default WebSocket port (override with ws_url / OPENALGO_WS_URL).
const maxInstances = 100

// wsPortFor maps a Flask port to its instance's WebSocket port.
func wsPortFor(flaskPort string) string {
	p, err := strconv.Atoi(flaskPort)
	if err != nil || p < defaultFlaskPort || p >= defaultFlaskPort+maxInstances {
		return DefaultWSPort
	}
	base, _ := strconv.Atoi(DefaultWSPort)
	return strconv.Itoa(base + p - defaultFlaskPort)
}

func (r *Resolved) HasCredentials() bool {
	return r.Source != SourceNone
}

func (r *Resolved) Validate() error {
	if !r.HasCredentials() {
		return fmt.Errorf("authentication required\nHint: run `openalgo profile login` or set OPENALGO_API_KEY")
	}
	return nil
}

func loadGlobalConfig() Config {
	var cfg Config
	path := filepath.Join(Dir(), "config.yaml")
	data, err := os.ReadFile(path)
	if err != nil {
		return cfg
	}
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to parse %s: %v\n", path, err)
	}
	return cfg
}

// LoadGlobalConfig returns the stored config.yaml settings (zero values when
// the file is absent).
func LoadGlobalConfig() *Config {
	cfg := loadGlobalConfig()
	return &cfg
}

func LoadProfileByName(name string) *Profile {
	p := loadProfile(name)
	return &p
}

func loadProfile(name string) Profile {
	var p Profile
	if ValidateProfileName(name) != nil {
		return p
	}
	path := profilePath(name)
	data, err := os.ReadFile(path)
	if err != nil {
		return p
	}
	if err := yaml.Unmarshal(data, &p); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to parse %s: %v\n", path, err)
	}
	return p
}

func SaveGlobalConfig(cfg *Config) error {
	dir := Dir()
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, "config.yaml"), data, 0o600)
}

func SaveProfile(name string, p *Profile) error {
	if err := ValidateProfileName(name); err != nil {
		return err
	}
	dir := filepath.Join(Dir(), "profiles")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	data, err := yaml.Marshal(p)
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, name+".yaml"), data, 0o600)
}

func DeleteProfile(name string) error {
	if err := ValidateProfileName(name); err != nil {
		return err
	}
	return os.Remove(profilePath(name))
}

func ListProfiles() ([]string, error) {
	entries, err := os.ReadDir(filepath.Join(Dir(), "profiles"))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var names []string
	for _, e := range entries {
		if !e.IsDir() && filepath.Ext(e.Name()) == ".yaml" {
			names = append(names, e.Name()[:len(e.Name())-5])
		}
	}
	return names, nil
}

// resolve returns the first non-empty value.
func resolve(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}
