package enums

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func TestCollect(t *testing.T) {
	dir := t.TempDir()
	src := `package conf

type ModeEnum int

const (
	ModeUnknown ModeEnum = iota // unknown
	ModeFast    ModeEnum = iota // fast
	ModeSafe                    // safe
	modeCount                   // count
)

type Color string

const (
	ColorRed  Color = "red"
	ColorBlue Color = "blue"
)

type Size int64

const (
	KB Size = 1 << 10
	MB Size = 1 << 20
)

type Opts struct{}

const OptsDefault Opts = 0
`
	if err := os.WriteFile(filepath.Join(dir, "conf.go"), []byte(src), 0o600); err != nil {
		t.Fatal(err)
	}
	// test files are not parsed
	if err := os.WriteFile(filepath.Join(dir, "conf_test.go"), []byte("package conf\n\nconst ModeTest ModeEnum = 9 // test\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	values, err := Collect([]string{dir})
	if err != nil {
		t.Fatalf("collect: %s", err)
	}
	for _, test := range []struct {
		name   string
		expect []string
	}{
		{"ModeEnum", []string{"fast", "safe"}},
		{"Color", []string{"red", "blue"}},
		{"Size", nil},
		{"Opts", nil},
	} {
		if got := values.Get("conf", test.name); !slices.Equal(got, test.expect) {
			t.Errorf("%s: expected %q, got %q", test.name, test.expect, got)
		}
	}
}
