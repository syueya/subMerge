package source

import "testing"

func TestFilterSkipsInfoNodesAndLoopback(t *testing.T) {
	if err := DefaultFilterError(); err != nil {
		t.Fatal(err)
	}
	cf, err := CompileFilter(DefaultFilterOptions())
	if err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		name   string
		server string
		keep   bool
	}{
		{"剩余流量：995.29 GB", "127.0.0.1", false},
		{"套餐到期：长期有效", "127.0.0.1", false},
		{"🇯🇵日本高速01|CTCU|0.5x", "cfyes.example.com", true},
		{"🇭🇰香港高速01|BGP", "aws-link1.example.com", true},
		{"官网地址", "1.2.3.4", false},
		{"normal-node", "localhost", false},
	}
	for _, tc := range cases {
		keep, _ := cf.ShouldKeep(ParsedProxy{Name: tc.name, Server: tc.server, Type: "vless", Port: 443})
		if keep != tc.keep {
			t.Fatalf("%s @ %s: keep=%v want %v", tc.name, tc.server, keep, tc.keep)
		}
	}
}

func TestFilterIncludeWhitelist(t *testing.T) {
	cf, err := CompileFilter(FilterOptions{
		IncludeNameRegex: `日本|香港|JP|HK`,
	})
	if err != nil {
		t.Fatal(err)
	}
	keep, _ := cf.ShouldKeep(ParsedProxy{Name: "🇺🇸美国01", Server: "a.com", Type: "ss", Port: 443})
	if keep {
		t.Fatal("US node should be excluded by include whitelist")
	}
	keep, _ = cf.ShouldKeep(ParsedProxy{Name: "🇯🇵日本01", Server: "a.com", Type: "ss", Port: 443})
	if !keep {
		t.Fatal("JP node should be kept")
	}
}

func TestCompileFilterInvalidRegex(t *testing.T) {
	_, err := CompileFilter(FilterOptions{ExcludeNameRegex: "("})
	if err == nil {
		t.Fatal("expected error")
	}
}
