package cmd

import (
	"regexp"
	"strings"
	"testing"

	"github.com/marketcalls/openalgo-cli/internal/api"
)

var kebabCase = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

// TestAllOpsValid iterates every generated Op via api.AllOps and validates
// summaries, flag descriptions, kebab-case names, and response field
// descriptions (the --schema comments).
func TestAllOpsValid(t *testing.T) {
	if len(api.AllOps) == 0 {
		t.Fatal("api.AllOps is empty - generator may not have run")
	}
	for _, op := range api.AllOps {
		if op.Summary == "" {
			t.Errorf("op %q has empty Summary", op.Name)
			continue
		}
		t.Run(op.Name, func(t *testing.T) {
			if op.Method != "GET" && op.Method != "POST" {
				t.Errorf("unexpected method %q", op.Method)
			}
			if !strings.HasPrefix(op.Path, "/") {
				t.Errorf("path %q is not relative to /api/v1", op.Path)
			}
			for _, f := range op.Flags {
				if f.Name == "" {
					t.Error("FlagDef has empty Name")
				}
				if f.Description == "" {
					t.Errorf("flag %q has empty Description", f.Name)
				}
				if !kebabCase.MatchString(f.Name) {
					t.Errorf("flag %q is not kebab-case", f.Name)
				}
				if f.OASName == "apikey" {
					t.Errorf("flag %q exposes apikey; the CLI injects it", f.Name)
				}
			}
			checkResponseFields(t, "", op.Response)
		})
	}
}

func checkResponseFields(t *testing.T, prefix string, fields []api.ResponseField) {
	t.Helper()
	for _, f := range fields {
		if f.Description == "" {
			t.Errorf("response field %s%s has empty Description", prefix, f.Name)
		}
		checkResponseFields(t, prefix+f.Name+".", f.Fields)
	}
}
