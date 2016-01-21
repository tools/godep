package main

import "testing"

func TestTrimGoVersion(t *testing.T) {
	var cases = []struct {
		in, out string
		err     bool
	}{
		{in: "go1.5", out: "go1.5", err: false},
		{in: "go1.5beta1", out: "go1.5", err: false},
		{in: "go1.5.1", out: "go1.5", err: false},
		{in: "devel", out: "devel", err: false},
		{in: "boom", out: "", err: true},
	}

	for _, c := range cases {
		mv, err := trimGoVersion(c.in)
		if err != nil && !c.err {
			t.Errorf("Unexpected error: %s", err)
		}
		if mv != c.out {
			t.Errorf("Expected trimGoVersion(%s) == '%s', but got '%s'", c.in, c.out, mv)
		}
	}
}
