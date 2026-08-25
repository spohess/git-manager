package task

import (
	"path/filepath"
	"testing"
)

func statusContext(path string) *Context {
	ctx := newContext(path)
	ctx.Opts = Options{Name: "status"}
	return ctx
}

func TestStatusArvoreLimpaEmDia(t *testing.T) {
	_, work := newSandbox(t)

	if err := runStatus(statusContext(work)); err != nil {
		t.Fatalf("status: %v", err)
	}
}

func TestStatusDetectaAlteracoesLocais(t *testing.T) {
	_, work := newSandbox(t)

	writeFile(t, work, "novo.txt", "trabalho em andamento\n")
	gitRun(t, work, "add", "novo.txt")
	writeFile(t, work, "solto.txt", "sem stage\n")

	ctx := statusContext(work)
	if err := runStatus(ctx); err != nil {
		t.Fatalf("status: %v", err)
	}
	if ctx.detail == "" {
		t.Fatal("esperado detalhe de status preenchido")
	}
	if status := gitRun(t, work, "status", "--porcelain"); status == "" {
		t.Fatal("sandbox deveria continuar com alterações não commitadas")
	}
}

func TestStatusComparaComPrincipalQuandoBranchDiferente(t *testing.T) {
	origin, work := newSandbox(t)

	gitRun(t, work, "checkout", "-b", "feature/x")
	writeFile(t, work, "feature.txt", "trabalho da feature\n")
	gitRun(t, work, "add", ".")
	gitRun(t, work, "commit", "-m", "feature")
	gitRun(t, work, "push", "--set-upstream", "origin", "feature/x")

	outro := filepath.Join(filepath.Dir(work), "outro")
	gitRun(t, filepath.Dir(work), "clone", origin, outro)
	gitRun(t, outro, "config", "user.email", "teste@exemplo.com")
	gitRun(t, outro, "config", "user.name", "Teste")
	writeFile(t, outro, "novo-na-main.txt", "vindo da main\n")
	gitRun(t, outro, "add", ".")
	gitRun(t, outro, "commit", "-m", "novidade na main")
	gitRun(t, outro, "push", "origin", "main")

	if err := runStatus(statusContext(work)); err != nil {
		t.Fatalf("status: %v", err)
	}

	if branch := gitRun(t, work, "rev-parse", "--abbrev-ref", "HEAD"); branch != "feature/x" {
		t.Errorf("branch atual = %q, esperado feature/x", branch)
	}
}
