package cmd

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestAPIPath(t *testing.T) {
	cases := map[string]string{
		"funds":                "/funds",
		"/funds":               "/funds",
		"/api/v1/funds":        "/funds",
		"api/v1/funds":         "/funds",
		"/api/v1/ticker/NSE:X": "/ticker/NSE:X",
		"/api/v1":              "/",
		"/api/v10/funds":       "/api/v10/funds",
	}
	for in, want := range cases {
		if got := apiPath(in); got != want {
			t.Errorf("apiPath(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestParseMethodPath(t *testing.T) {
	cases := []struct {
		args      []string
		queryOnly bool
		method    string
		path      string
	}{
		{[]string{"/funds"}, false, "POST", "/funds"},
		{[]string{"/instruments"}, true, "GET", "/instruments"},
		{[]string{"get", "/instruments"}, false, "GET", "/instruments"},
		{[]string{"POST", "/placeorder"}, true, "POST", "/placeorder"},
		{[]string{"DELETE", "/x"}, false, "DELETE", "/x"},
	}
	for _, tc := range cases {
		m, p := parseMethodPath(tc.args, tc.queryOnly)
		if m != tc.method || p != tc.path {
			t.Errorf("parseMethodPath(%v, %v) = %s %s, want %s %s", tc.args, tc.queryOnly, m, p, tc.method, tc.path)
		}
	}
}

func TestDecodeJSONObject(t *testing.T) {
	m, err := decodeJSONObject([]byte(`{"quantity": 10, "price": 0.05}`))
	if err != nil {
		t.Fatal(err)
	}
	if got := marshal(t, m); got != `{"price":0.05,"quantity":10}` {
		t.Errorf("round trip = %s", got)
	}
	for _, bad := range []string{`[1]`, `null`, `{`, `"s"`, `{"a":1} trailing`, `{"a":1}{"b":2}`} {
		if _, err := decodeJSONObject([]byte(bad)); err == nil {
			t.Errorf("decodeJSONObject(%s) should fail", bad)
		}
	}
}

func TestReadImplicitStdin_SilentPipeDoesNotBlock(t *testing.T) {
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = r.Close(); _ = w.Close() }()
	start := time.Now()
	b, err := readImplicitStdin(r, 50*time.Millisecond)
	if err != nil || b != nil {
		t.Errorf("silent pipe: b = %q, err = %v", b, err)
	}
	if time.Since(start) > 2*time.Second {
		t.Error("readImplicitStdin blocked on a silent pipe")
	}
}

func TestReadImplicitStdin_ReadsPipedBody(t *testing.T) {
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = r.Close() }()
	go func() {
		_, _ = w.Write([]byte(`{"symbol":"SBIN"}`))
		_ = w.Close()
	}()
	b, err := readImplicitStdin(r, 2*time.Second)
	if err != nil || string(b) != `{"symbol":"SBIN"}` {
		t.Errorf("b = %q, err = %v", b, err)
	}
}

func TestReadBodyFlag_File(t *testing.T) {
	path := filepath.Join(t.TempDir(), "body.json")
	if err := os.WriteFile(path, []byte(`{"a":1}`), 0o600); err != nil {
		t.Fatal(err)
	}
	b, err := readBodyFlag("@" + path)
	if err != nil || string(b) != `{"a":1}` {
		t.Errorf("b = %q, err = %v", b, err)
	}
	if b, _ := readBodyFlag(`{"b":2}`); string(b) != `{"b":2}` {
		t.Errorf("literal body = %q", b)
	}
}
