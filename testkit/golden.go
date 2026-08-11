package testkit

import (
	"fmt"
	"os"
	"path/filepath"
)

type GoldenMismatchError struct {
	Path string
	Want string
	Got  string
}

func (e *GoldenMismatchError) Error() string {
	return fmt.Sprintf("golden mismatch at %s: want %q, got %q", e.Path, e.Want, e.Got)
}

// CompareGolden compares a snapshot with a golden file. When UPDATE_GOLDEN=1,
// it writes the current snapshot instead.
func CompareGolden(path, snapshot string) error {
	if os.Getenv("UPDATE_GOLDEN") == "1" {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return err
		}
		return os.WriteFile(path, []byte(snapshot), 0o644)
	}
	want, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if string(want) != snapshot {
		return &GoldenMismatchError{Path: path, Want: string(want), Got: snapshot}
	}
	return nil
}
