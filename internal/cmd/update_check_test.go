package cmd

import (
	"net/http"
	"net/http/httptest"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestInstallMethodFor(t *testing.T) {
	cases := []struct {
		path, gobin, gopath, want string
	}{
		{"/opt/homebrew/Cellar/openalgo-cli/0.1.0/bin/openalgo", "", "/home/u/go", installHomebrew},
		{"/home/u/go/bin/openalgo", "", "/home/u/go", installGoInstall},
		{"/custom/bin/openalgo", "/custom/bin", "/home/u/go", installGoInstall},
		{"/usr/local/bin/openalgo", "", "/home/u/go", installScript},
		{"/home/u/.local/bin/openalgo", "", "/home/u/go", installScript},
		{"/home/u/gobin/openalgo", "/home/u/go", "/home/u/go", installScript},
	}
	if runtime.GOOS == "windows" {
		cases = append(cases, []struct {
			path, gobin, gopath, want string
		}{
			{`C:\Users\u\go\bin\openalgo.exe`, "", `C:\Users\u\go`, installGoInstall},
			{`D:\tools\bin\openalgo.exe`, `D:\tools\bin`, `C:\Users\u\go`, installGoInstall},
			{`C:\Users\u\AppData\Local\Programs\openalgo\openalgo.exe`, "", `C:\Users\u\go`, installScript},
		}...)
	}
	for _, c := range cases {
		if got := installMethodFor(c.path, c.gobin, c.gopath); got != c.want {
			t.Errorf("installMethodFor(%q) = %q, want %q", c.path, got, c.want)
		}
	}
}

// TestScriptUpgradeCommand checks that a script install is upgraded in place
// (OPENALGO_INSTALL_DIR pinned to the binary's directory), never via go install.
func TestScriptUpgradeCommand(t *testing.T) {
	got := scriptUpgradeCommand("linux", "/usr/local/bin")
	if !strings.Contains(got, "install.sh") || !strings.Contains(got, "OPENALGO_INSTALL_DIR='/usr/local/bin' sh") {
		t.Errorf("unix = %q", got)
	}
	got = scriptUpgradeCommand("windows", `C:\Users\u\AppData\Local\Programs\openalgo`)
	if !strings.Contains(got, "install.ps1") || !strings.Contains(got, `$env:OPENALGO_INSTALL_DIR='C:\Users\u\AppData\Local\Programs\openalgo'`) {
		t.Errorf("windows = %q", got)
	}
	if got := scriptUpgradeCommand("darwin", "/it's/bin"); !strings.Contains(got, `'/it'\''s/bin'`) {
		t.Errorf("quoting = %q", got)
	}
}

func TestLatestFromRedirect(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "https://github.com/marketcalls/openalgo-cli/releases/tag/v9.8.7", http.StatusFound)
	}))
	defer srv.Close()
	orig := releasesLatestURL
	releasesLatestURL = srv.URL
	defer func() { releasesLatestURL = orig }()

	got, err := latestFromRedirect(5 * time.Second)
	if err != nil || got != "v9.8.7" {
		t.Errorf("latestFromRedirect = %q, %v; want v9.8.7", got, err)
	}
}

func TestWithoutEnv(t *testing.T) {
	got := withoutEnv([]string{"PATH=/bin", "PSModulePath=x", "psmodulepath=y", "HOME=/h"}, "PSModulePath")
	if strings.Join(got, ",") != "PATH=/bin,HOME=/h" {
		t.Errorf("withoutEnv = %v", got)
	}
}
