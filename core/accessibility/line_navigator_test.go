package accessibility

import "testing"

func TestLineNavigator(t *testing.T) {
	text := "button#btn Label: Click\ncheckbox#chk Label: Check state: checked\nprogress Label: Load value: 50%\n"
	nav := NewLineNavigator(text)

	if nav.Current() != "button#btn Label: Click" {
		t.Errorf("Current = %q", nav.Current())
	}
	if nav.Next() != "checkbox#chk Label: Check state: checked" {
		t.Errorf("Next = %q", nav.Current())
	}
	if nav.Next() != "progress Label: Load value: 50%" {
		t.Errorf("Next = %q", nav.Current())
	}
	if nav.Next() != "" {
		t.Errorf("Next = %q, expected empty", nav.Current())
	}
	if nav.Previous() != "progress Label: Load value: 50%" {
		t.Errorf("Previous = %q", nav.Current())
	}
	if nav.AnnounceCurrent() != "Screen reader announcement: progress Label: Load value: 50%" {
		t.Errorf("Announce = %q", nav.AnnounceCurrent())
	}
}
