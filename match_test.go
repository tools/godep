package main

import "testing"

func TestMatchPattern(t *testing.T) {
	// Test cases from $GOROOT/src/cmd/go/match_test.go.
	cases := []struct {
		pat  string
		path string
		want bool
	}{
		{"...", "foo", true},
		{"net", "net", true},
		{"net", "net/http", false},
		{"net/http", "net", false},
		{"net/http", "net/http", true},
		{"net...", "netchan", true},
		{"net...", "net", true},
		{"net...", "net/http", true},
		{"net...", "not/http", false},
		{"net/...", "netchan", false},
		{"net/...", "net", true},
		{"net/...", "net/http", true},
		{"net/...", "not/http", false},
	}
	for _, test := range cases {
		ok := matchPattern(test.pat)(test.path)
		if ok != test.want {
			t.Errorf("matchPackages(%q)(%q) = %v want %v", test.pat, test.path, ok, test.want)
		}
	}
}
