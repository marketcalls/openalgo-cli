// Package useragent builds the CLI's User-Agent header:
// OpenAlgo-CLI/<version> <OS>/<arch>.
package useragent

import (
	"fmt"
	"runtime"
)

// Build returns the User-Agent string for the given CLI version, e.g.
// "OpenAlgo-CLI/0.0.1 darwin/arm64".
func Build(version string) string {
	return fmt.Sprintf("OpenAlgo-CLI/%s %s/%s", version, runtime.GOOS, runtime.GOARCH)
}
