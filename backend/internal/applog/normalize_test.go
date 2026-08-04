package applog

import "testing"

func TestNormalizeOutputDefaultBoth(t *testing.T) {
	if got := NormalizeOutput(""); got != "both" {
		t.Fatalf("empty => %q want both", got)
	}
	if got := NormalizeOutput("console"); got != "console" {
		t.Fatalf("console => %q", got)
	}
	if got := NormalizeOutput("weird"); got != "both" {
		t.Fatalf("unknown => %q want both", got)
	}
}
