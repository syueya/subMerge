package updater

import "testing"

func TestParseVersionAndString(t *testing.T) {
	for _, raw := range []string{"1.0.0", "v1.2.3", "2.0.0-rc.1+build.5", "0.0.0-alpha"} {
		version, err := ParseVersion(raw)
		if err != nil {
			t.Fatalf("ParseVersion(%q): %v", raw, err)
		}
		want := raw
		if raw[0] == 'v' {
			want = raw[1:]
		}
		if version.String() != want {
			t.Fatalf("ParseVersion(%q).String() = %q, want %q", raw, version.String(), want)
		}
	}
}

func TestParseVersionRejectsInvalid(t *testing.T) {
	invalid := []string{
		"", "v", "1", "1.2", "1.2.3.4", "01.2.3", "1.02.3", "1.2.03",
		"1.2.x", "1.2.3-", "1.2.3+", "1.2.3-alpha..1", "1.2.3-01",
		"1.2.3-alpha!", "1.2.3+a+b",
	}
	for _, raw := range invalid {
		if _, err := ParseVersion(raw); err == nil {
			t.Errorf("ParseVersion(%q) unexpectedly succeeded", raw)
		}
	}
}

func TestVersionPrecedenceFromSemVerSpecification(t *testing.T) {
	ordered := []string{
		"1.0.0-alpha",
		"1.0.0-alpha.1",
		"1.0.0-alpha.beta",
		"1.0.0-beta",
		"1.0.0-beta.2",
		"1.0.0-beta.11",
		"1.0.0-rc.1",
		"1.0.0",
		"1.0.1",
		"1.1.0",
		"2.0.0",
	}
	for index := 0; index < len(ordered)-1; index++ {
		comparison, err := CompareVersions(ordered[index], ordered[index+1])
		if err != nil {
			t.Fatal(err)
		}
		if comparison >= 0 {
			t.Errorf("expected %s < %s", ordered[index], ordered[index+1])
		}
	}
	comparison, err := CompareVersions("1.2.3+build.1", "1.2.3+build.2")
	if err != nil || comparison != 0 {
		t.Fatalf("build metadata changed precedence: comparison=%d err=%v", comparison, err)
	}
	comparison, err = CompareVersions("1.0.0-999999999999999999999999", "1.0.0-1000000000000000000000000")
	if err != nil || comparison >= 0 {
		t.Fatalf("unbounded numeric prerelease comparison=%d err=%v", comparison, err)
	}
	comparison, err = CompareVersions("184467440737095516160.0.0", "184467440737095516161.0.0")
	if err != nil || comparison >= 0 {
		t.Fatalf("unbounded core comparison=%d err=%v", comparison, err)
	}
}
