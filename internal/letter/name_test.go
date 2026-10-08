package letter

import (
	"testing"

	"github.com/nitrocode/breakup/internal/state"
)

func TestNormalizeName(t *testing.T) {
	tests := []struct {
		in      string
		want    string
		wantErr bool
	}{
		{"June", "june", false},
		{"  MARA  ", "mara", false},
		{"o'Hara", "o'hara", false},
		{"mary-jane", "mary-jane", false},
		{"", "", true},
		{"x", "", true},
		{"mary jane", "", true},
		{"breakup", "", true},
		{"please", "", true},
		{"june!", "", true},
		{"thisnameiswaytoolong", "", true},
	}
	for _, tt := range tests {
		got, err := NormalizeName(tt.in)
		if tt.wantErr {
			if err == nil {
				t.Errorf("NormalizeName(%q) err=nil, want error", tt.in)
			}
			continue
		}
		if err != nil {
			t.Errorf("NormalizeName(%q): %v", tt.in, err)
			continue
		}
		if got != tt.want {
			t.Errorf("NormalizeName(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestTitle(t *testing.T) {
	if Title(nil) != "breakup" {
		t.Fatal("nil")
	}
	if Title(&state.State{}) != "breakup" {
		t.Fatal("empty")
	}
	if Title(&state.State{Name: "june"}) != "june" {
		t.Fatal("named")
	}
}
