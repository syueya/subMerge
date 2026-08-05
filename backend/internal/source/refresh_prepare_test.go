package source

import (
	"strings"
	"testing"
)

func TestPrepareProxiesLimitsFilteredNameSamples(t *testing.T) {
	filter, err := CompileFilter(FilterOptions{ExcludeNameRegex: ".*"})
	if err != nil {
		t.Fatal(err)
	}
	proxies := make([]ParsedProxy, 0, maxFilteredNameSamples+2)
	for i := 0; i < maxFilteredNameSamples+2; i++ {
		name := "node-" + string(rune('a'+i%26))
		if i == 0 {
			name = "node\nwith\tcontrols"
		}
		proxies = append(proxies, ParsedProxy{Name: name, Type: "ss", Server: "example.com", Port: 443})
	}

	stats, err := prepareProxies(1, "source", "auto", "US", proxies, filter)
	if err != nil {
		t.Fatal(err)
	}
	if stats.filteredTotal != len(proxies) {
		t.Fatalf("filtered total = %d, want %d", stats.filteredTotal, len(proxies))
	}
	if len(stats.filteredNames) != maxFilteredNameSamples {
		t.Fatalf("filtered sample count = %d, want %d", len(stats.filteredNames), maxFilteredNameSamples)
	}
	if omitted := stats.filteredTotal - len(stats.filteredNames); omitted != 2 {
		t.Fatalf("omitted = %d, want 2", omitted)
	}
	if strings.ContainsAny(stats.filteredNames[0], "\r\n\t") {
		t.Fatalf("filtered name contains control characters: %q", stats.filteredNames[0])
	}
}

func TestPrepareProxiesCollectsRegionConflicts(t *testing.T) {
	filter, err := CompileFilter(FilterOptions{})
	if err != nil {
		t.Fatal(err)
	}
	stats, err := prepareProxies(1, "source", "auto", "UNK", []ParsedProxy{
		{Name: "🇨🇳台湾高速01", Type: "vless", Server: "example.com", Port: 443},
	}, filter)
	if err != nil {
		t.Fatal(err)
	}
	if len(stats.kept) != 1 || stats.kept[0].region != "TW" {
		t.Fatalf("prepared region = %q, stats=%+v", stats.kept[0].region, stats)
	}
	if stats.regionConflictTotal != 1 || len(stats.regionConflicts) != 1 {
		t.Fatalf("conflicts = %d samples=%d", stats.regionConflictTotal, len(stats.regionConflicts))
	}
	if got := stats.regionConflicts[0]; got.FlagRegion != "CN" || got.KeywordRegion != "TW" || got.ResolvedRegion != "TW" {
		t.Fatalf("conflict = %+v", got)
	}
}

func TestCleanFilteredNameTruncatesLongNames(t *testing.T) {
	name := cleanFilteredName(strings.Repeat("中", maxFilteredNameLen+20))
	if got := len([]rune(name)); got != maxFilteredNameLen+3 {
		t.Fatalf("cleaned name rune length = %d, want %d", got, maxFilteredNameLen+3)
	}
	if !strings.HasSuffix(name, "...") {
		t.Fatalf("cleaned name should have truncation suffix: %q", name)
	}
}
