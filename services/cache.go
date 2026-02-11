package services

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/mitchellh/go-homedir"
	"gopkg.in/yaml.v3"

	"github.com/AxeForging/yankrun/domain"
)

func cachePath() (string, error) {
	home, err := homedir.Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".yankrun", "cache.yaml"), nil
}

// LoadCacheFrom loads cache from a specific path
func LoadCacheFrom(path string) (*domain.Cache, error) {
	cache := &domain.Cache{}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cache, nil
		}
		return cache, err
	}
	if err := yaml.Unmarshal(data, cache); err != nil {
		return &domain.Cache{}, nil
	}
	return cache, nil
}

// SaveCacheTo saves cache to a specific path
func SaveCacheTo(path string, cache *domain.Cache) error {
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	enc := yaml.NewEncoder(f)
	enc.SetIndent(2)
	return enc.Encode(cache)
}

// LoadCache loads cache from the default path (~/.yankrun/cache.yaml)
func LoadCache() (*domain.Cache, error) {
	path, err := cachePath()
	if err != nil {
		return &domain.Cache{}, err
	}
	return LoadCacheFrom(path)
}

// SaveCache saves cache to the default path (~/.yankrun/cache.yaml)
func SaveCache(cache *domain.Cache) error {
	path, err := cachePath()
	if err != nil {
		return err
	}
	return SaveCacheTo(path, cache)
}

// GitHubConfigSHA returns a deterministic hash of the GitHub config for cache invalidation
func GitHubConfigSHA(gh domain.GitHubConfig) string {
	h := sha256.New()
	h.Write([]byte(gh.User))
	h.Write([]byte{0})
	orgs := make([]string, len(gh.Orgs))
	copy(orgs, gh.Orgs)
	sort.Strings(orgs)
	for _, o := range orgs {
		h.Write([]byte(o))
		h.Write([]byte{0})
	}
	h.Write([]byte(gh.Topic))
	h.Write([]byte{0})
	h.Write([]byte(gh.Prefix))
	h.Write([]byte{0})
	if gh.IncludePrivate {
		h.Write([]byte("1"))
	} else {
		h.Write([]byte("0"))
	}
	return fmt.Sprintf("%x", h.Sum(nil))
}

// LookupVars finds cached variables for a given URL and branch
func LookupVars(cache *domain.Cache, url, branch string) (*domain.CachedTemplateVars, bool) {
	for i := range cache.TemplateVars {
		if cache.TemplateVars[i].URL == url && cache.TemplateVars[i].Branch == branch {
			return &cache.TemplateVars[i], true
		}
	}
	return nil, false
}

// UpdateVars updates or adds cached variables for a given URL and branch
func UpdateVars(cache *domain.Cache, url, branch, sha string, vars map[string]int) {
	for i := range cache.TemplateVars {
		if cache.TemplateVars[i].URL == url && cache.TemplateVars[i].Branch == branch {
			cache.TemplateVars[i].SHA = sha
			cache.TemplateVars[i].Variables = vars
			return
		}
	}
	cache.TemplateVars = append(cache.TemplateVars, domain.CachedTemplateVars{
		URL:       url,
		Branch:    branch,
		SHA:       sha,
		Variables: vars,
	})
}
