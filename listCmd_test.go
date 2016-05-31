package main

import "testing"

func TestListCmd(t *testing.T) {
	var cases = []struct {
		cwd      string
		args     []string
		flagR    bool
		flagT    bool
		vendor   bool
		start    []*node
		altstart []*node
		want     string
		werr     bool
	}{
		{ // 0 - simple case, one dependency
			cwd:    "C",
			vendor: true,
			start: []*node{
				{
					"C",
					"",
					[]*node{
						{"main.go", pkg("main", "D"), nil},
						{"+git", "", nil},
					},
				},
				{
					"D",
					"",
					[]*node{
						{"main.go", pkg("D"), nil},
						{"+git", "D1", nil},
					},
				},
			},
			want: ``,
		},
	}

	for range cases {

	}

}
