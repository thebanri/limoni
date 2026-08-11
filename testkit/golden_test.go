package testkit

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestCompareGolden(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "screen.txt")
	if err := os.WriteFile(path, []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := CompareGolden(path, "hello"); err != nil {
		t.Fatal(err)
	}
	if err := CompareGolden(path, "different"); err == nil {
		t.Fatal("expected golden mismatch")
	} else {
		var mismatch *GoldenMismatchError
		if !errors.As(err, &mismatch) || mismatch.Path != path {
			t.Fatalf("mismatch error = %v", err)
		}
	}
}
