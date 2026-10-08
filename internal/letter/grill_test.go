package letter

import "testing"

func TestIsBeg(t *testing.T) {
	tests := []struct {
		in   string
		want bool
	}{
		{"please", true},
		{"Please let me back", true},
		{"i'm begging you", true},
		{"im begging", true},
		{"pls", false},
		{"plz", false},
		{"sorry", false},
		{"", false},
		{"the bug is hard", false},
		{"pretty please", true},
		{"pleased to meet you", false},
	}
	for _, tt := range tests {
		if got := IsBeg(tt.in); got != tt.want {
			t.Errorf("IsBeg(%q) = %v, want %v", tt.in, got, tt.want)
		}
	}
}

func TestGiveUpTurnNeedsThreeThenPlease(t *testing.T) {
	ctx := Context{}
	_, ok := GiveUpTurn(1, true, ctx)
	if ok {
		t.Fatal("please should not skip round 1")
	}
	_, ok = GiveUpTurn(2, true, ctx)
	if ok {
		t.Fatal("please should not skip round 2")
	}
	body, ok := GiveUpTurn(3, false, ctx)
	if ok {
		t.Fatal("round 3 without please should fail")
	}
	if !contains(body, "--please") {
		t.Errorf("should demand --please, got %s", body)
	}
	body, ok = GiveUpTurn(3, true, ctx)
	if !ok {
		t.Fatal("round 3 with please should pass")
	}
	if !contains(body, "we're back") {
		t.Errorf("got %s", body)
	}
}

func TestGrillRoastMentionsTheWork(t *testing.T) {
	ctx := Context{}
	body := GrillRoastWhy("i'm stuck on a bug", ctx)
	if !contains(body, "work") && !contains(body, "hard") {
		t.Errorf("expected a roast of the work excuse: %s", body)
	}
	if !contains(body, "what did you try") {
		t.Errorf("expected next question: %s", body)
	}
}

func TestGrillBegNoOnPls(t *testing.T) {
	body := GrillBegNo("pls", Context{})
	if !contains(body, "pls") && !contains(body, "spell") {
		t.Errorf("got %s", body)
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub ||
		(func() bool {
			for i := 0; i+len(sub) <= len(s); i++ {
				if s[i:i+len(sub)] == sub {
					return true
				}
			}
			return false
		})())
}
