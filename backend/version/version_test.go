package version

import "testing"

func TestStringReturnsLinkedValue(t *testing.T) {
	original := Value
	t.Cleanup(func() { Value = original })

	Value = "1.2.3"
	if got := String(); got != "1.2.3" {
		t.Fatalf("String() = %q, want linked value", got)
	}
}
