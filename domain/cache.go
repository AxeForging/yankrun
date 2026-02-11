package domain

type Cache struct {
	GitHubConfigSHA string               `yaml:"github_config_sha"`
	GitHubRepos     []TemplateRepo       `yaml:"github_repos"`
	TemplateVars    []CachedTemplateVars `yaml:"template_vars"`
}

type CachedTemplateVars struct {
	URL       string         `yaml:"url"`
	Branch    string         `yaml:"branch"`
	SHA       string         `yaml:"sha"`
	Variables map[string]int `yaml:"variables"`
}
