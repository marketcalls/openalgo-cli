package cmd

import (
	"strings"
	"testing"
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
