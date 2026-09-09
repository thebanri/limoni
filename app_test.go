package limoni

import (
	"testing"
)

func TestAppOptions(t *testing.T) {
	// Default options
	var defaultCfg appConfig
	if defaultCfg.catchCtrlC {
		t.Fatalf("expected catchCtrlC to default to false")
	}

	// WithCatchCtrlC(true)
	var cfg1 appConfig
	opt1 := WithCatchCtrlC(true)
	opt1(&cfg1)
	if !cfg1.catchCtrlC {
		t.Fatalf("expected catchCtrlC to be true")
	}

	// WithoutDefaultQuitKeys()
	var cfg2 appConfig
	opt2 := WithoutDefaultQuitKeys()
	opt2(&cfg2)
	if !cfg2.catchCtrlC {
		t.Fatalf("expected WithoutDefaultQuitKeys to enable catchCtrlC")
	}
}
