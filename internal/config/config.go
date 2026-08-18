package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

type Project struct {
	Name         string `yaml:"name"`
	Path         string `yaml:"path"`
	Main         bool   `yaml:"main"`
	CommitSufixo string `yaml:"commit-sufixo"`
}

type Config struct {
	Projects []Project `yaml:"projects"`
}

const envPath = "GIT_MANAGER_CONFIG"

func candidatePaths() []string {
	paths := []string{
		"git-manager.yml",
		"git-manager.yaml",
		"projects.yml",
		"projects.yaml",
		"config.yml",
		"config.yaml",
	}
	if home, err := os.UserHomeDir(); err == nil {
		paths = append(paths,
			filepath.Join(home, ".config", "git-manager", "config.yml"),
			filepath.Join(home, ".config", "git-manager", "config.yaml"),
			filepath.Join(home, ".git-manager.yml"),
			filepath.Join(home, ".git-manager.yaml"),
		)
	}
	return paths
}

func Load(path string) (*Config, string, error) {
	resolved, err := resolvePath(path)
	if err != nil {
		return nil, "", err
	}
	data, err := os.ReadFile(resolved)
	if err != nil {
		return nil, "", fmt.Errorf("não foi possível ler %s: %w", resolved, err)
	}
	cfg, err := Parse(data)
	if err != nil {
		return nil, resolved, fmt.Errorf("%s: %w", resolved, err)
	}
	return cfg, resolved, nil
}

func Parse(data []byte) (*Config, error) {
	var cfg Config
	decoder := yaml.NewDecoder(strings.NewReader(string(data)))
	decoder.KnownFields(true)
	if err := decoder.Decode(&cfg); err != nil {
		return nil, fmt.Errorf("yaml inválido: %w", err)
	}
	if len(cfg.Projects) == 0 {
		return nil, fmt.Errorf("nenhum projeto declarado na chave \"projects\"")
	}
	seen := make(map[string]bool, len(cfg.Projects))
	for i := range cfg.Projects {
		p := &cfg.Projects[i]
		p.Name = strings.TrimSpace(p.Name)
		p.CommitSufixo = strings.TrimSpace(p.CommitSufixo)
		if p.Name == "" {
			return nil, fmt.Errorf("projeto #%d: chave \"name\" é obrigatória", i+1)
		}
		if seen[p.Name] {
			return nil, fmt.Errorf("projeto %q está duplicado", p.Name)
		}
		seen[p.Name] = true
		if strings.TrimSpace(p.Path) == "" {
			return nil, fmt.Errorf("projeto %q: chave \"path\" é obrigatória", p.Name)
		}
		expanded, err := expand(p.Path)
		if err != nil {
			return nil, fmt.Errorf("projeto %q: %w", p.Name, err)
		}
		p.Path = expanded
	}
	return &cfg, nil
}

func Select(cfg *Config, name string, includeNonMain bool) ([]Project, error) {
	name = strings.TrimSpace(name)
	if name != "" {
		return selectByNames(cfg, name)
	}
	var selected []Project
	for _, p := range cfg.Projects {
		if includeNonMain || p.Main {
			selected = append(selected, p)
		}
	}
	if len(selected) == 0 {
		return nil, fmt.Errorf("nenhum projeto com \"main: true\"; use --no-main para incluir todos ou --project=nome")
	}
	return selected, nil
}

func selectByNames(cfg *Config, raw string) ([]Project, error) {
	var selected []Project
	seen := make(map[string]bool)
	for _, part := range strings.Split(raw, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		key := strings.ToLower(part)
		if seen[key] {
			continue
		}
		project, ok := findProject(cfg, part)
		if !ok {
			return nil, fmt.Errorf("projeto %q não encontrado na configuração (disponíveis: %s)", part, strings.Join(names(cfg), ", "))
		}
		seen[key] = true
		selected = append(selected, project)
	}
	if len(selected) == 0 {
		return nil, fmt.Errorf("nenhum projeto informado em --project")
	}
	return selected, nil
}

func findProject(cfg *Config, name string) (Project, bool) {
	for _, p := range cfg.Projects {
		if strings.EqualFold(p.Name, name) {
			return p, true
		}
	}
	return Project{}, false
}

func names(cfg *Config) []string {
	list := make([]string, 0, len(cfg.Projects))
	for _, p := range cfg.Projects {
		list = append(list, p.Name)
	}
	return list
}

func resolvePath(path string) (string, error) {
	if strings.TrimSpace(path) != "" {
		expanded, err := expand(path)
		if err != nil {
			return "", err
		}
		if _, err := os.Stat(expanded); err != nil {
			return "", fmt.Errorf("arquivo de configuração não encontrado: %s", expanded)
		}
		return expanded, nil
	}
	if fromEnv := strings.TrimSpace(os.Getenv(envPath)); fromEnv != "" {
		return resolvePath(fromEnv)
	}
	for _, candidate := range candidatePaths() {
		expanded, err := expand(candidate)
		if err != nil {
			continue
		}
		if info, err := os.Stat(expanded); err == nil && !info.IsDir() {
			return expanded, nil
		}
	}
	return "", fmt.Errorf("arquivo de configuração não encontrado; informe --config=caminho.yml, defina %s ou crie um dos arquivos: %s", envPath, strings.Join(candidatePaths(), ", "))
}

func expand(path string) (string, error) {
	path = strings.TrimSpace(path)
	if strings.HasPrefix(path, "~") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("não foi possível resolver \"~\" em %s: %w", path, err)
		}
		path = filepath.Join(home, strings.TrimPrefix(path, "~"))
	}
	path = os.ExpandEnv(path)
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("caminho inválido %s: %w", path, err)
	}
	return abs, nil
}
