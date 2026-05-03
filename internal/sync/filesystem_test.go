package sync

import (
	"reflect"
	"sort"
	"testing"

	"github.com/salingsing5/coolisap/internal/azuredevops"
)

func TestSanitizeTitle(t *testing.T) {
	cases := map[string]string{
		"Getting Started":    "Getting Started",
		"Foo/Bar":            "Foo_Bar",
		`a:b*c?d"e<f>g|h\i`:  "a_b_c_d_e_f_g_h_i",
		"":                   "_",
		".":                  "_.",
		"..":                 "_..",
		"Foo.":                "Foo_",
		"Foo ":                "Foo_",
		"Foo. ":               "Foo._", // the loop stops at the first non-trailing-dot-or-space
		"CON":                 "CON_",
		"con":                 "con_",
		"PRN.md":              "PRN.md_", // extension doesn't save it
		"COM1":                "COM1_",
		"COM10":               "COM10", // only 1-9 are reserved
		"console":             "console",
	}
	for in, want := range cases {
		if got := SanitizeTitle(in); got != want {
			t.Errorf("SanitizeTitle(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestWalkTree(t *testing.T) {
	root := &azuredevops.Page{Path: "/", SubPages: []azuredevops.Page{
		{ID: 1, Path: "/Alpha", Content: "A", SubPages: []azuredevops.Page{
			{ID: 2, Path: "/Alpha/Beta", Content: "B"},
		}},
		{ID: 3, Path: "/Bad:Name", Content: "X"},
	}}
	got := WalkTree(root)
	sort.Slice(got, func(i, j int) bool { return got[i].RelPath < got[j].RelPath })
	want := []FileWrite{
		{RelPath: "Alpha.md", Content: "A", WikiPath: "/Alpha", PageID: 1},
		{RelPath: "Alpha/Beta.md", Content: "B", WikiPath: "/Alpha/Beta", PageID: 2},
		{RelPath: "Bad_Name.md", Content: "X", WikiPath: "/Bad:Name", PageID: 3},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %+v, want %+v", got, want)
	}
}

func TestWalkTreeNil(t *testing.T) {
	if got := WalkTree(nil); len(got) != 0 {
		t.Fatalf("WalkTree(nil) = %+v, want empty", got)
	}
}
