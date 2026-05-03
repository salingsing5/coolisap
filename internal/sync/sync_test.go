package sync

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/salingsing5/coolisap/internal/azuredevops"
)

type fakeFetcher struct{ tree *azuredevops.Page }

func (f *fakeFetcher) GetWikiPageTree(_ context.Context, _, _ string) (*azuredevops.Page, error) {
	return f.tree, nil
}

func (f *fakeFetcher) GetWikiPage(_ context.Context, _, _, _ string) (*azuredevops.Page, error) {
	return &azuredevops.Page{ID: 999}, nil
}

func (f *fakeFetcher) GetWikiInfo(_ context.Context, _, _ string) (*azuredevops.WikiInfo, error) {
	return &azuredevops.WikiInfo{RepositoryID: "fake-repo-id"}, nil
}

func (f *fakeFetcher) GetWikiAttachment(_ context.Context, _, _, _ string) ([]byte, error) {
	return nil, nil
}

const (
	testOrg     = "contoso"
	testProject = "Platform"
	testWiki    = "Platform.wiki"
)

func testOpts(f Fetcher, dir string) Options {
	return Options{
		Fetcher:      f,
		Organization: testOrg,
		Project:      testProject,
		Wiki:         testWiki,
		OutputDir:    dir,
	}
}

func TestRunWritesAndPrunes(t *testing.T) {
	dir := t.TempDir()

	first := &fakeFetcher{tree: &azuredevops.Page{Path: "/", SubPages: []azuredevops.Page{
		{ID: 1, Path: "/A", Content: "a"},
		{ID: 2, Path: "/B", Content: "b"},
	}}}
	if _, err := Run(context.Background(), testOpts(first, dir)); err != nil {
		t.Fatal(err)
	}

	second := &fakeFetcher{tree: &azuredevops.Page{Path: "/", SubPages: []azuredevops.Page{
		{ID: 1, Path: "/A", Content: "a-updated"},
		{ID: 3, Path: "/C", Content: "c"},
	}}}
	res, err := Run(context.Background(), testOpts(second, dir))
	if err != nil {
		t.Fatal(err)
	}
	if res.Written != 2 || res.Deleted != 1 {
		t.Fatalf("counts: written=%d deleted=%d, want 2/1", res.Written, res.Deleted)
	}
	if _, err := os.Stat(filepath.Join(dir, "B.md")); !os.IsNotExist(err) {
		t.Fatalf("B.md should be pruned")
	}
	body, _ := os.ReadFile(filepath.Join(dir, "A.md"))
	wantBody := "a-updated" + wikiSourceFooter(testOrg, testProject, testWiki, "/A", 1)
	if string(body) != wantBody {
		t.Fatalf("A.md = %q, want %q", body, wantBody)
	}
}

func TestRunWritesNestedDirs(t *testing.T) {
	dir := t.TempDir()
	f := &fakeFetcher{tree: &azuredevops.Page{Path: "/", SubPages: []azuredevops.Page{
		{ID: 10, Path: "/Home", Content: "h", SubPages: []azuredevops.Page{
			{ID: 11, Path: "/Home/Child", Content: "c"},
		}},
	}}}
	if _, err := Run(context.Background(), testOpts(f, dir)); err != nil {
		t.Fatal(err)
	}
	for _, rel := range []string{"Home.md", "Home/Child.md", ".wikisync.json"} {
		if _, err := os.Stat(filepath.Join(dir, rel)); err != nil {
			t.Errorf("missing %s: %v", rel, err)
		}
	}
}

func TestRunRemovesEmptiedDirs(t *testing.T) {
	dir := t.TempDir()
	first := &fakeFetcher{tree: &azuredevops.Page{Path: "/", SubPages: []azuredevops.Page{
		{ID: 10, Path: "/Home", Content: "h", SubPages: []azuredevops.Page{
			{ID: 11, Path: "/Home/Child", Content: "c"},
		}},
	}}}
	if _, err := Run(context.Background(), testOpts(first, dir)); err != nil {
		t.Fatal(err)
	}
	second := &fakeFetcher{tree: &azuredevops.Page{Path: "/"}}
	if _, err := Run(context.Background(), testOpts(second, dir)); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "Home")); !os.IsNotExist(err) {
		t.Errorf("Home/ should be pruned once empty; got %v", err)
	}
}

func TestRunRequiresOrganization(t *testing.T) {
	f := &fakeFetcher{tree: &azuredevops.Page{Path: "/"}}
	_, err := Run(context.Background(), Options{Fetcher: f, OutputDir: t.TempDir()})
	if err == nil || !strings.Contains(err.Error(), "Organization is required") {
		t.Fatalf("want 'Organization is required' error, got %v", err)
	}
}

func TestRunAppendsSourceFooter(t *testing.T) {
	dir := t.TempDir()
	f := &fakeFetcher{tree: &azuredevops.Page{Path: "/", SubPages: []azuredevops.Page{
		{ID: 204, Path: "/Overview of Database Tables and Scripts", Content: "# Body"},
	}}}
	if _, err := Run(context.Background(), testOpts(f, dir)); err != nil {
		t.Fatal(err)
	}
	body, err := os.ReadFile(filepath.Join(dir, "Overview of Database Tables and Scripts.md"))
	if err != nil {
		t.Fatal(err)
	}
	wantURL := "https://dev.azure.com/contoso/Platform/_wiki/wikis/Platform.wiki/204"
	wantTitle := "Overview of Database Tables and Scripts"
	if !strings.Contains(string(body), "**Source:** ["+wantTitle+"]("+wantURL+")") {
		t.Fatalf("footer missing or malformed:\n%s", body)
	}
	if !strings.HasPrefix(string(body), "# Body") {
		t.Fatalf("body should start with original content, got:\n%s", body)
	}
}

func TestWikiPageURL(t *testing.T) {
	cases := []struct {
		name                       string
		org, project, wiki         string
		id                         int64
		want                       string
	}{
		{
			name:    "project with space gets escaped",
			org:     "ajtickets",
			project: "AJ Tickets",
			wiki:    "AJ-Tickets.wiki",
			id:      204,
			want:    "https://dev.azure.com/ajtickets/AJ%20Tickets/_wiki/wikis/AJ-Tickets.wiki/204",
		},
		{
			name:    "wiki with space also escaped",
			org:     "contoso",
			project: "Platform",
			wiki:    "My Wiki.wiki",
			id:      1,
			want:    "https://dev.azure.com/contoso/Platform/_wiki/wikis/My%20Wiki.wiki/1",
		},
		{
			name:    "large ID",
			org:     "o",
			project: "p",
			wiki:    "w",
			id:      9223372036854775807,
			want:    "https://dev.azure.com/o/p/_wiki/wikis/w/9223372036854775807",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := wikiPageURL(c.org, c.project, c.wiki, c.id)
			if got != c.want {
				t.Errorf("\n got: %s\nwant: %s", got, c.want)
			}
		})
	}
}

func TestRunFillSetsSubpageID(t *testing.T) {
	// Subpage has empty Content + zero ID in the tree, so fill must
	// fetch via GetWikiPage and copy both fields. The fake returns ID 999.
	dir := t.TempDir()
	f := &fakeFetcher{tree: &azuredevops.Page{Path: "/", SubPages: []azuredevops.Page{
		{ID: 5, Path: "/Parent", Content: "p", SubPages: []azuredevops.Page{
			{Path: "/Parent/Child"}, // no ID, no Content
		}},
	}}}
	if _, err := Run(context.Background(), testOpts(f, dir)); err != nil {
		t.Fatal(err)
	}
	body, err := os.ReadFile(filepath.Join(dir, "Parent", "Child.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(body), "/Platform.wiki/999") {
		t.Fatalf("subpage URL should embed fetched ID 999; got:\n%s", body)
	}
}
