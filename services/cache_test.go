package services

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/AxeForging/yankrun/domain"
)

func TestLoadCacheFrom_NonExistent(t *testing.T) {
	cache, err := LoadCacheFrom("/nonexistent/path/cache.yaml")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cache.TemplateVars) != 0 {
		t.Errorf("expected empty cache, got %d template vars", len(cache.TemplateVars))
	}
	if len(cache.GitHubRepos) != 0 {
		t.Errorf("expected empty GitHub repos, got %d", len(cache.GitHubRepos))
	}
}

func TestLoadCacheFrom_Corrupt(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "cache.yaml")
	if err := os.WriteFile(path, []byte("{{invalid yaml"), 0644); err != nil {
		t.Fatal(err)
	}
	cache, err := LoadCacheFrom(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cache.TemplateVars) != 0 {
		t.Errorf("expected empty cache on corrupt file, got %d template vars", len(cache.TemplateVars))
	}
}

func TestSaveCacheAndLoad(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "cache.yaml")

	cache := &domain.Cache{
		GitHubConfigSHA: "abc123",
		GitHubRepos: []domain.TemplateRepo{
			{Name: "test/repo", URL: "https://github.com/test/repo.git", DefaultBranch: "main"},
		},
		TemplateVars: []domain.CachedTemplateVars{
			{URL: "https://github.com/test/repo.git", Branch: "main", SHA: "def456", Variables: map[string]int{"APP_NAME": 5, "VERSION": 2}},
		},
	}

	if err := SaveCacheTo(path, cache); err != nil {
		t.Fatalf("SaveCacheTo failed: %v", err)
	}

	loaded, err := LoadCacheFrom(path)
	if err != nil {
		t.Fatalf("LoadCacheFrom failed: %v", err)
	}

	if loaded.GitHubConfigSHA != "abc123" {
		t.Errorf("expected GitHubConfigSHA abc123, got %s", loaded.GitHubConfigSHA)
	}
	if len(loaded.GitHubRepos) != 1 {
		t.Fatalf("expected 1 GitHub repo, got %d", len(loaded.GitHubRepos))
	}
	if loaded.GitHubRepos[0].Name != "test/repo" {
		t.Errorf("expected repo name test/repo, got %s", loaded.GitHubRepos[0].Name)
	}
	if loaded.GitHubRepos[0].DefaultBranch != "main" {
		t.Errorf("expected default branch main, got %s", loaded.GitHubRepos[0].DefaultBranch)
	}
	if len(loaded.TemplateVars) != 1 {
		t.Fatalf("expected 1 template vars entry, got %d", len(loaded.TemplateVars))
	}
	if loaded.TemplateVars[0].SHA != "def456" {
		t.Errorf("expected SHA def456, got %s", loaded.TemplateVars[0].SHA)
	}
	if loaded.TemplateVars[0].Variables["APP_NAME"] != 5 {
		t.Errorf("expected APP_NAME count 5, got %d", loaded.TemplateVars[0].Variables["APP_NAME"])
	}
	if loaded.TemplateVars[0].Variables["VERSION"] != 2 {
		t.Errorf("expected VERSION count 2, got %d", loaded.TemplateVars[0].Variables["VERSION"])
	}
}

func TestSaveCacheTo_CreatesDir(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sub", "dir", "cache.yaml")

	cache := &domain.Cache{GitHubConfigSHA: "test"}
	if err := SaveCacheTo(path, cache); err != nil {
		t.Fatalf("SaveCacheTo failed to create parent dirs: %v", err)
	}

	loaded, err := LoadCacheFrom(path)
	if err != nil {
		t.Fatalf("LoadCacheFrom failed: %v", err)
	}
	if loaded.GitHubConfigSHA != "test" {
		t.Errorf("expected SHA test, got %s", loaded.GitHubConfigSHA)
	}
}

func TestGitHubConfigSHA_Deterministic(t *testing.T) {
	cfg1 := domain.GitHubConfig{User: "alice", Orgs: []string{"org1"}}
	cfg2 := domain.GitHubConfig{User: "alice", Orgs: []string{"org1"}}

	sha1 := GitHubConfigSHA(cfg1)
	sha2 := GitHubConfigSHA(cfg2)

	if sha1 != sha2 {
		t.Errorf("same config should produce same SHA: %s vs %s", sha1, sha2)
	}
}

func TestGitHubConfigSHA_DifferentUsers(t *testing.T) {
	cfg1 := domain.GitHubConfig{User: "alice", Orgs: []string{"org1"}}
	cfg2 := domain.GitHubConfig{User: "bob", Orgs: []string{"org1"}}

	if GitHubConfigSHA(cfg1) == GitHubConfigSHA(cfg2) {
		t.Error("different users should produce different SHA")
	}
}

func TestGitHubConfigSHA_OrgOrderIndependent(t *testing.T) {
	cfg1 := domain.GitHubConfig{User: "alice", Orgs: []string{"org2", "org1"}}
	cfg2 := domain.GitHubConfig{User: "alice", Orgs: []string{"org1", "org2"}}

	sha1 := GitHubConfigSHA(cfg1)
	sha2 := GitHubConfigSHA(cfg2)

	if sha1 != sha2 {
		t.Errorf("org order should not matter: %s vs %s", sha1, sha2)
	}
}

func TestGitHubConfigSHA_IncludePrivate(t *testing.T) {
	cfg1 := domain.GitHubConfig{User: "alice", IncludePrivate: false}
	cfg2 := domain.GitHubConfig{User: "alice", IncludePrivate: true}

	if GitHubConfigSHA(cfg1) == GitHubConfigSHA(cfg2) {
		t.Error("include_private change should produce different SHA")
	}
}

func TestGitHubConfigSHA_TopicAndPrefix(t *testing.T) {
	cfg1 := domain.GitHubConfig{User: "alice", Topic: "template"}
	cfg2 := domain.GitHubConfig{User: "alice", Topic: "boilerplate"}

	if GitHubConfigSHA(cfg1) == GitHubConfigSHA(cfg2) {
		t.Error("different topics should produce different SHA")
	}

	cfg3 := domain.GitHubConfig{User: "alice", Prefix: "tpl-"}
	cfg4 := domain.GitHubConfig{User: "alice", Prefix: "tmpl-"}

	if GitHubConfigSHA(cfg3) == GitHubConfigSHA(cfg4) {
		t.Error("different prefixes should produce different SHA")
	}
}

func TestLookupVars_Found(t *testing.T) {
	cache := &domain.Cache{
		TemplateVars: []domain.CachedTemplateVars{
			{URL: "https://github.com/test/a.git", Branch: "main", SHA: "aaa", Variables: map[string]int{"FOO": 1}},
			{URL: "https://github.com/test/b.git", Branch: "dev", SHA: "bbb", Variables: map[string]int{"BAR": 2}},
		},
	}

	v, ok := LookupVars(cache, "https://github.com/test/a.git", "main")
	if !ok {
		t.Fatal("expected to find cached vars")
	}
	if v.SHA != "aaa" {
		t.Errorf("expected SHA aaa, got %s", v.SHA)
	}
	if v.Variables["FOO"] != 1 {
		t.Errorf("expected FOO=1, got %d", v.Variables["FOO"])
	}
}

func TestLookupVars_NotFound(t *testing.T) {
	cache := &domain.Cache{
		TemplateVars: []domain.CachedTemplateVars{
			{URL: "https://github.com/test/a.git", Branch: "main", SHA: "aaa", Variables: map[string]int{"FOO": 1}},
		},
	}

	// Wrong branch
	_, ok := LookupVars(cache, "https://github.com/test/a.git", "dev")
	if ok {
		t.Error("should not find vars for wrong branch")
	}

	// Wrong URL
	_, ok = LookupVars(cache, "https://github.com/test/c.git", "main")
	if ok {
		t.Error("should not find vars for wrong URL")
	}
}

func TestLookupVars_EmptyCache(t *testing.T) {
	cache := &domain.Cache{}

	_, ok := LookupVars(cache, "https://github.com/test/a.git", "main")
	if ok {
		t.Error("should not find vars in empty cache")
	}
}

func TestUpdateVars_NewEntry(t *testing.T) {
	cache := &domain.Cache{}

	UpdateVars(cache, "https://github.com/test/a.git", "main", "aaa", map[string]int{"FOO": 3})

	if len(cache.TemplateVars) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(cache.TemplateVars))
	}
	if cache.TemplateVars[0].URL != "https://github.com/test/a.git" {
		t.Errorf("unexpected URL: %s", cache.TemplateVars[0].URL)
	}
	if cache.TemplateVars[0].Branch != "main" {
		t.Errorf("unexpected branch: %s", cache.TemplateVars[0].Branch)
	}
	if cache.TemplateVars[0].SHA != "aaa" {
		t.Errorf("unexpected SHA: %s", cache.TemplateVars[0].SHA)
	}
	if cache.TemplateVars[0].Variables["FOO"] != 3 {
		t.Errorf("expected FOO=3, got %d", cache.TemplateVars[0].Variables["FOO"])
	}
}

func TestUpdateVars_ExistingEntry(t *testing.T) {
	cache := &domain.Cache{
		TemplateVars: []domain.CachedTemplateVars{
			{URL: "https://github.com/test/a.git", Branch: "main", SHA: "old", Variables: map[string]int{"FOO": 1}},
		},
	}

	UpdateVars(cache, "https://github.com/test/a.git", "main", "new", map[string]int{"FOO": 5, "BAR": 2})

	if len(cache.TemplateVars) != 1 {
		t.Fatalf("expected 1 entry (updated), got %d", len(cache.TemplateVars))
	}
	if cache.TemplateVars[0].SHA != "new" {
		t.Errorf("expected SHA new, got %s", cache.TemplateVars[0].SHA)
	}
	if cache.TemplateVars[0].Variables["FOO"] != 5 {
		t.Errorf("expected FOO=5, got %d", cache.TemplateVars[0].Variables["FOO"])
	}
	if cache.TemplateVars[0].Variables["BAR"] != 2 {
		t.Errorf("expected BAR=2, got %d", cache.TemplateVars[0].Variables["BAR"])
	}
}

func TestUpdateVars_MultipleEntries(t *testing.T) {
	cache := &domain.Cache{}

	UpdateVars(cache, "https://github.com/test/a.git", "main", "aaa", map[string]int{"FOO": 1})
	UpdateVars(cache, "https://github.com/test/b.git", "dev", "bbb", map[string]int{"BAR": 2})
	UpdateVars(cache, "https://github.com/test/a.git", "main", "ccc", map[string]int{"FOO": 3})

	if len(cache.TemplateVars) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(cache.TemplateVars))
	}
	// First entry should be updated
	if cache.TemplateVars[0].SHA != "ccc" {
		t.Errorf("expected SHA ccc for first entry, got %s", cache.TemplateVars[0].SHA)
	}
	// Second entry should be unchanged
	if cache.TemplateVars[1].SHA != "bbb" {
		t.Errorf("expected SHA bbb for second entry, got %s", cache.TemplateVars[1].SHA)
	}
}

func TestCacheRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "cache.yaml")

	cache := &domain.Cache{}
	UpdateVars(cache, "git@github.com:org/repo.git", "main", "abc123", map[string]int{"APP_NAME": 10, "VERSION": 3})
	cache.GitHubConfigSHA = GitHubConfigSHA(domain.GitHubConfig{User: "testuser", Orgs: []string{"org1"}})
	cache.GitHubRepos = []domain.TemplateRepo{
		{Name: "org1/template-go", URL: "git@github.com:org1/template-go.git", Description: "Go template", DefaultBranch: "main"},
	}

	if err := SaveCacheTo(path, cache); err != nil {
		t.Fatalf("save failed: %v", err)
	}

	loaded, err := LoadCacheFrom(path)
	if err != nil {
		t.Fatalf("load failed: %v", err)
	}

	// Verify GitHub repos
	if len(loaded.GitHubRepos) != 1 || loaded.GitHubRepos[0].Name != "org1/template-go" {
		t.Errorf("GitHub repos mismatch: %+v", loaded.GitHubRepos)
	}

	// Verify template vars via lookup
	v, ok := LookupVars(loaded, "git@github.com:org/repo.git", "main")
	if !ok {
		t.Fatal("expected to find cached vars after round-trip")
	}
	if v.SHA != "abc123" {
		t.Errorf("expected SHA abc123, got %s", v.SHA)
	}
	if v.Variables["APP_NAME"] != 10 || v.Variables["VERSION"] != 3 {
		t.Errorf("unexpected variables: %v", v.Variables)
	}
}
