package config

import (
	"os"
	"path/filepath"
	"testing"
)

const sample = `projects:
  - name: backend
    path: /tmp/projects/backend
    main: true
    commit-sufixo: teste-teste-teste

  - name: admin
    path: /tmp/projects/frontend-admin
    main: true

  - name: client
    path: /tmp/projects/frontend-client
    main: false
`

func TestParse(t *testing.T) {
	cfg, err := Parse([]byte(sample))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(cfg.Projects) != 3 {
		t.Fatalf("esperado 3 projetos, obtido %d", len(cfg.Projects))
	}
	if cfg.Projects[0].CommitSufixo != "teste-teste-teste" {
		t.Errorf("commit-sufixo não carregado: %q", cfg.Projects[0].CommitSufixo)
	}
	if cfg.Projects[1].CommitSufixo != "" {
		t.Errorf("commit-sufixo deveria ser opcional, obtido %q", cfg.Projects[1].CommitSufixo)
	}
	if cfg.Projects[2].Main {
		t.Errorf("projeto client deveria ter main: false")
	}
}

func TestParseInvalid(t *testing.T) {
	cases := map[string]string{
		"sem projetos":   "projects: []\n",
		"sem nome":       "projects:\n  - path: /tmp/a\n    main: true\n",
		"sem path":       "projects:\n  - name: a\n    main: true\n",
		"nome repetido":  "projects:\n  - name: a\n    path: /tmp/a\n  - name: a\n    path: /tmp/b\n",
		"chave inválida": "projects:\n  - name: a\n    path: /tmp/a\n    commit_sufixo: x\n",
	}
	for name, data := range cases {
		if _, err := Parse([]byte(data)); err == nil {
			t.Errorf("%s: esperado erro", name)
		}
	}
}

func TestParseExpandsHome(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("sem HOME")
	}
	cfg, err := Parse([]byte("projects:\n  - name: a\n    path: ~/projetos/a\n    main: true\n"))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	want := filepath.Join(home, "projetos", "a")
	if cfg.Projects[0].Path != want {
		t.Errorf("path = %q, esperado %q", cfg.Projects[0].Path, want)
	}
}

func TestSelect(t *testing.T) {
	cfg, err := Parse([]byte(sample))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	onlyMain, err := Select(cfg, "", false)
	if err != nil {
		t.Fatalf("Select: %v", err)
	}
	if len(onlyMain) != 2 {
		t.Errorf("sem --no-main esperado 2 projetos, obtido %d", len(onlyMain))
	}

	all, err := Select(cfg, "", true)
	if err != nil {
		t.Fatalf("Select: %v", err)
	}
	if len(all) != 3 {
		t.Errorf("com --no-main esperado 3 projetos, obtido %d", len(all))
	}

	single, err := Select(cfg, "client", false)
	if err != nil {
		t.Fatalf("Select: %v", err)
	}
	if len(single) != 1 || single[0].Name != "client" {
		t.Errorf("--project deveria selecionar o projeto mesmo com main: false, obtido %+v", single)
	}

	if _, err := Select(cfg, "inexistente", true); err == nil {
		t.Error("esperado erro para projeto inexistente")
	}
}

func TestSelectMultiplosProjetos(t *testing.T) {
	cfg, err := Parse([]byte(sample))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	multi, err := Select(cfg, "client, backend", false)
	if err != nil {
		t.Fatalf("Select: %v", err)
	}
	if len(multi) != 2 {
		t.Fatalf("esperado 2 projetos, obtido %d", len(multi))
	}
	if multi[0].Name != "client" || multi[1].Name != "backend" {
		t.Errorf("ordem inesperada: %+v", multi)
	}

	dedup, err := Select(cfg, "backend,backend", false)
	if err != nil {
		t.Fatalf("Select: %v", err)
	}
	if len(dedup) != 1 {
		t.Errorf("nomes repetidos deveriam ser deduplicados, obtido %d", len(dedup))
	}

	if _, err := Select(cfg, "backend,inexistente", true); err == nil {
		t.Error("esperado erro quando um dos projetos não existe")
	}
}

func TestLoadFromFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "git-manager.yml")
	if err := os.WriteFile(path, []byte(sample), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, resolved, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if resolved != path {
		t.Errorf("resolved = %q, esperado %q", resolved, path)
	}
	if len(cfg.Projects) != 3 {
		t.Errorf("esperado 3 projetos, obtido %d", len(cfg.Projects))
	}
	if _, _, err := Load(filepath.Join(dir, "nao-existe.yml")); err == nil {
		t.Error("esperado erro para arquivo inexistente")
	}
}
