package main

import (
	"path/filepath"
	"reflect"
	"testing"
)

func TestWithOwnDir(t *testing.T) {
	// Absolute on every platform, so that the binary's own directory is one
	// the test can name: "/h/go/bin" is not absolute on Windows.
	root := t.TempDir()
	dir := filepath.Join(root, "go", "bin")
	other := filepath.Join(root, "usr", "bin")
	exe := filepath.Join(dir, "backdrop-shell")
	sep := string(filepath.ListSeparator)

	for _, c := range []struct {
		name string
		env  []string
		exe  string
		want []string
	}{
		{"missing from PATH", []string{"HOME=/h", "PATH=" + other}, exe,
			[]string{"HOME=/h", "PATH=" + other + sep + dir}},
		{"already there", []string{"PATH=" + dir + sep + other}, exe,
			[]string{"PATH=" + dir + sep + other}},
		{"empty PATH", []string{"PATH="}, exe, []string{"PATH=" + dir}},
		{"no PATH", []string{"HOME=/h"}, exe, []string{"HOME=/h", "PATH=" + dir}},
		{"unknown binary", []string{"PATH=" + other}, "", []string{"PATH=" + other}},
	} {
		env := append([]string(nil), c.env...)
		if got := withOwnDir(env, c.exe); !reflect.DeepEqual(got, c.want) {
			t.Errorf("%s: got %q, want %q", c.name, got, c.want)
		}
		if !reflect.DeepEqual(env, c.env) {
			t.Errorf("%s: the environment passed in was changed to %q", c.name, env)
		}
	}
}
