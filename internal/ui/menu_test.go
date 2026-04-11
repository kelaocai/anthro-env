package ui

import "testing"

func TestParseMenuSelection(t *testing.T) {
	cases := []struct {
		name    string
		in      string
		max     int
		want    int
		wantErr bool
	}{
		{"empty string returns 0", "", 3, 0, false},
		{"spaces only returns 0", "   ", 3, 0, false},
		{"zero selection", "0", 3, 0, false},
		{"valid selection", "2", 3, 2, false},
		{"invalid non-numeric", "x", 3, 0, true},
		{"out of range high", "4", 3, 0, true},
		{"out of range negative", "-1", 3, 0, true},
		{"max boundary valid", "3", 3, 3, false},
		{"max boundary invalid", "4", 3, 0, true},
		{"whitespace padded", " 2 ", 3, 2, false},
		{"large input", "9999999999", 3, 0, true},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := ParseMenuSelection(c.in, c.max)
			if (err != nil) != c.wantErr {
				t.Fatalf("in=%q err=%v wantErr=%v", c.in, err, c.wantErr)
			}
			if got != c.want {
				t.Fatalf("in=%q got=%d want=%d", c.in, got, c.want)
			}
		})
	}
}

// TestMenuSelectionRange tests selection range logic
func TestMenuSelectionRange(t *testing.T) {
	// Simulates the selection logic in runMenuInteractive
	type scenario struct {
		name     string
		selected int
		total    int
		wantUp   bool
		wantDown bool
	}

	profiles := []string{"a", "b", "c"}
	total := len(profiles) + 1 // +1 for Exit option

	cases := []scenario{
		{"Exit selected, up disabled", 0, total, false, true},
		{"First profile, up enabled (to Exit)", 1, total, true, true},
		{"Middle profile, both enabled", 2, total, true, true},
		{"Last profile, down disabled", 3, total, true, false},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			canMoveUp := c.selected > 0
			canMoveDown := c.selected < c.total-1

			if canMoveUp != c.wantUp {
				t.Errorf("selected=%d total=%d canMoveUp=%v wantUp=%v",
					c.selected, c.total, canMoveUp, c.wantUp)
			}
			if canMoveDown != c.wantDown {
				t.Errorf("selected=%d total=%d canMoveDown=%v wantDown=%v",
					c.selected, c.total, canMoveDown, c.wantDown)
			}
		})
	}
}

// TestDefaultSelection tests that current profile is selected by default
func TestDefaultSelection(t *testing.T) {
	profiles := []string{"default", "work", "home"}
	active := "work"

	// Simulates the default selection logic
	selected := 0
	if active != "" {
		for i, p := range profiles {
			if p == active {
				selected = i + 1
				break
			}
		}
	}

	want := 2 // "work" is at index 1, so selected = 1+1 = 2
	if selected != want {
		t.Errorf("active=%q selected=%d want=%d", active, selected, want)
	}
}

// TestDefaultSelectionNoActive tests default selection when no profile is active
func TestDefaultSelectionNoActive(t *testing.T) {
	profiles := []string{"default", "work", "home"}
	active := ""

	selected := 0
	if active != "" {
		for i, p := range profiles {
			if p == active {
				selected = i + 1
				break
			}
		}
	}

	// Should remain 0 (Exit) when no active profile
	if selected != 0 {
		t.Errorf("active=%q selected=%d want=0", active, selected)
	}
}

// TestProfileIndexConversion tests conversion between selection and profile index
func TestProfileIndexConversion(t *testing.T) {
	profiles := []string{"alpha", "beta", "gamma"}

	cases := []struct {
		name      string
		selected  int // selection number (0 = Exit, 1 = first profile)
		wantIndex int // profile index in slice (-1 if Exit)
		wantName  string
	}{
		{"Exit", 0, -1, ""},
		{"first_profile", 1, 0, "alpha"},
		{"second_profile", 2, 1, "beta"},
		{"third_profile", 3, 2, "gamma"},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if c.selected == 0 {
				if c.wantIndex != -1 {
					t.Errorf("selected=0 wantIndex=-1 got=%d", c.wantIndex)
				}
				return
			}
			gotIndex := c.selected - 1
			if gotIndex < 0 || gotIndex >= len(profiles) {
				t.Errorf("selected=%d out of range", c.selected)
				return
			}
			if profiles[gotIndex] != c.wantName {
				t.Errorf("profiles[%d]=%q want=%q", gotIndex, profiles[gotIndex], c.wantName)
			}
		})
	}
}
