package cmd

import (
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
)

const (
	installHomebrew  = "homebrew"
	installGoInstall = "goinstall"
	// installScript is a release binary placed by install.sh or install.ps1
	// (or copied by hand) anywhere outside Homebrew and GOPATH/GOBIN.
	installScript = "script"
)

const (
	installShURL  = "https://raw.githubusercontent.com/" + repoOwner + "/" + repoName + "/main/install.sh"
	installPs1URL = "https://raw.githubusercontent.com/" + repoOwner + "/" + repoName + "/main/install.ps1"
)

// goosWindows is runtime.GOOS on Windows.
const goosWindows = "windows"

func detectInstallMethod() string {
	exe, err := os.Executable()
	if err != nil {
		return installScript
	}
	resolved, err := filepath.EvalSymlinks(exe)
	if err != nil {
		resolved = exe
	}
	return installMethodFor(resolved, os.Getenv("GOBIN"), os.Getenv("GOPATH"))
}

// installMethodFor classifies a resolved binary path. Anything that is not
// under Homebrew or a Go bin directory came from a release archive, so it
// is upgraded by re-running the install script.
//
// Paths are compared with forward slashes so the same rules hold for
// Windows paths (C:\Users\u\go\bin\openalgo.exe).
func installMethodFor(resolved, gobin, gopath string) string {
	resolved = filepath.ToSlash(resolved)
	if strings.Contains(resolved, "/Cellar/") || strings.Contains(resolved, "/homebrew/") {
		return installHomebrew
	}
	if gobin != "" && strings.HasPrefix(resolved, strings.TrimSuffix(filepath.ToSlash(gobin), "/")+"/") {
		return installGoInstall
	}
	if gopath == "" {
		home, _ := os.UserHomeDir()
		gopath = filepath.Join(home, "go")
	}
	for _, p := range filepath.SplitList(gopath) {
		if strings.HasPrefix(resolved, strings.TrimSuffix(filepath.ToSlash(p), "/")+"/bin/") {
			return installGoInstall
		}
	}
	return installScript
}

func upgradeCommand(method string) string {
	switch method {
	case installHomebrew:
		return "brew upgrade marketcalls/tap/openalgo-cli"
	case installGoInstall:
		return "go install github.com/marketcalls/openalgo-cli/cmd/openalgo@latest"
	default:
		return scriptUpgradeCommand(runtime.GOOS, installDir())
	}
}

// installDir is the directory of the running binary, so the install script
// replaces it in place instead of shadowing it with a second copy.
func installDir() string {
	exe, err := os.Executable()
	if err != nil {
		return ""
	}
	if resolved, err := filepath.EvalSymlinks(exe); err == nil {
		exe = resolved
	}
	return filepath.Dir(exe)
}

// scriptUpgradeCommand re-runs the platform's install script with
// OPENALGO_INSTALL_DIR pinned to dir.
func scriptUpgradeCommand(goos, dir string) string {
	if goos == goosWindows {
		cmd := "irm " + installPs1URL + " | iex"
		if dir != "" {
			cmd = "$env:OPENALGO_INSTALL_DIR='" + strings.ReplaceAll(dir, "'", "''") + "'; " + cmd
		}
		return cmd
	}
	cmd := "curl -fsSL " + installShURL + " | "
	if dir != "" {
		cmd += "OPENALGO_INSTALL_DIR='" + strings.ReplaceAll(dir, "'", `'\''`) + "' "
	}
	return cmd + "sh"
}

// versionNewer reports whether latest is strictly greater than current
// using numeric major.minor.patch comparison.
func versionNewer(latest, current string) bool {
	parseVer := func(s string) [3]int {
		s = strings.TrimPrefix(s, "v")
		if idx := strings.IndexByte(s, '-'); idx != -1 {
			s = s[:idx]
		}
		parts := strings.SplitN(s, ".", 3)
		var v [3]int
		for i := 0; i < len(parts) && i < 3; i++ {
			n, _ := strconv.Atoi(parts[i])
			v[i] = n
		}
		return v
	}
	l, c := parseVer(latest), parseVer(current)
	for i := range 3 {
		if l[i] != c[i] {
			return l[i] > c[i]
		}
	}
	return false
}
