package testkit

import (
	"os"
	"path/filepath"
)

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
		return os.ErrInvalid
	}
	return nil
}
