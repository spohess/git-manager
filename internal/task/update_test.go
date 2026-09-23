package task

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"git-manager/internal/config"
	"git-manager/internal/git"
)

func gitRun(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s em %s: %v\n%s", strings.Join(args, " "), dir, err, out)
	}
	return strings.TrimSpace(string(out))
}

func writeFile(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func newSandbox(t *testing.T) (origin string, work string) {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git indisponível")
	}
	root := t.TempDir()
	origin = filepath.Join(root, "origin.git")
	gitRun(t, root, "init", "--bare", "--initial-branch=main", origin)

	work = filepath.Join(root, "work")
	gitRun(t, root, "clone", origin, work)
	gitRun(t, work, "config", "user.email", "teste@exemplo.com")
	gitRun(t, work, "config", "user.name", "Teste")
	gitRun(t, work, "config", "commit.gpgsign", "false")

	writeFile(t, work, "README.md", "inicial\n")
	gitRun(t, work, "add", ".")
	gitRun(t, work, "commit", "-m", "inicial")
	gitRun(t, work, "push", "--set-upstream", "origin", "main")
	return origin, work
}

func newContext(path string) *Context {
	return &Context{
		Project: config.Project{Name: "teste", Path: path, Main: true},
		Git:     git.New(path, false),
		Opts:    Options{Name: "update", Source: "main"},
	}
}

func TestUpdateCommitaEEnviaAlteracoesPendentes(t *testing.T) {
	_, work := newSandbox(t)

	gitRun(t, work, "checkout", "-b", "feature/x")
	writeFile(t, work, "novo.txt", "trabalho em andamento\n")

	if err := runUpdate(newContext(work)); err != nil {
		t.Fatalf("update: %v", err)
	}

	if branch := gitRun(t, work, "rev-parse", "--abbrev-ref", "HEAD"); branch != "main" {
		t.Errorf("branch atual = %q, esperado main", branch)
	}
	if status := gitRun(t, work, "status", "--porcelain"); status != "" {
		t.Errorf("árvore suja após o update: %q", status)
	}
	subject := gitRun(t, work, "log", "-1", "--format=%s", "origin/feature/x")
	if subject != autoCommitMessage {
		t.Errorf("mensagem do commit automático = %q, esperado %q", subject, autoCommitMessage)
	}
	files := gitRun(t, work, "show", "--name-only", "--format=", "origin/feature/x")
	if !strings.Contains(files, "novo.txt") {
		t.Errorf("commit automático não contém novo.txt: %q", files)
	}
}

func TestUpdateResetaMainComAlteracoesLocais(t *testing.T) {
	origin, work := newSandbox(t)

	outro := filepath.Join(filepath.Dir(work), "outro")
	gitRun(t, filepath.Dir(work), "clone", origin, outro)
	gitRun(t, outro, "config", "user.email", "teste@exemplo.com")
	gitRun(t, outro, "config", "user.name", "Teste")
	writeFile(t, outro, "remoto.txt", "vindo do remoto\n")
	gitRun(t, outro, "add", ".")
	gitRun(t, outro, "commit", "-m", "remoto")
	gitRun(t, outro, "push", "origin", "main")

	writeFile(t, work, "README.md", "alteração local que bloqueia o pull\n")

	if err := runUpdate(newContext(work)); err != nil {
		t.Fatalf("update: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(work, "README.md"))
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != "inicial\n" {
		t.Errorf("README.md = %q, esperado o conteúdo do remoto", content)
	}
	if _, err := os.Stat(filepath.Join(work, "remoto.txt")); err != nil {
		t.Errorf("commit remoto não foi trazido: %v", err)
	}
}

func TestUpdateFalhaSeBranchDestinoNaoExisteNoRemoto(t *testing.T) {
	_, work := newSandbox(t)

	gitRun(t, work, "checkout", "-b", "feature/x")
	writeFile(t, work, "novo.txt", "trabalho em andamento\n")

	ctx := newContext(work)
	ctx.Opts.Source = "develop"
	err := runUpdate(ctx)
	if err == nil || !strings.Contains(err.Error(), "develop") {
		t.Fatalf("esperado erro de branch de origem inexistente, obtido %v", err)
	}

	if branch := gitRun(t, work, "rev-parse", "--abbrev-ref", "HEAD"); branch != "feature/x" {
		t.Errorf("branch atual = %q, esperado feature/x", branch)
	}
	if status := gitRun(t, work, "status", "--porcelain"); !strings.Contains(status, "novo.txt") {
		t.Errorf("as alterações locais deveriam continuar pendentes: %q", status)
	}
}

func TestUpdateUsaBranchDestinoInformada(t *testing.T) {
	origin, work := newSandbox(t)

	gitRun(t, work, "checkout", "-b", "develop")
	writeFile(t, work, "develop.txt", "base do develop\n")
	gitRun(t, work, "add", ".")
	gitRun(t, work, "commit", "-m", "develop")
	gitRun(t, work, "push", "--set-upstream", "origin", "develop")
	gitRun(t, work, "checkout", "main")

	outro := filepath.Join(filepath.Dir(work), "outro")
	gitRun(t, filepath.Dir(work), "clone", origin, outro)
	gitRun(t, outro, "config", "user.email", "teste@exemplo.com")
	gitRun(t, outro, "config", "user.name", "Teste")
	gitRun(t, outro, "checkout", "develop")
	writeFile(t, outro, "remoto.txt", "vindo do remoto\n")
	gitRun(t, outro, "add", ".")
	gitRun(t, outro, "commit", "-m", "remoto")
	gitRun(t, outro, "push", "origin", "develop")

	ctx := newContext(work)
	ctx.Opts.Source = "develop"
	if err := runUpdate(ctx); err != nil {
		t.Fatalf("update: %v", err)
	}

	if branch := gitRun(t, work, "rev-parse", "--abbrev-ref", "HEAD"); branch != "develop" {
		t.Errorf("branch atual = %q, esperado develop", branch)
	}
	if _, err := os.Stat(filepath.Join(work, "remoto.txt")); err != nil {
		t.Errorf("commit remoto do develop não foi trazido: %v", err)
	}
}

func TestNewCriaBranchAposUpdate(t *testing.T) {
	_, work := newSandbox(t)

	ctx := newContext(work)
	ctx.Opts = Options{Name: "new", Branch: "feature/login", Source: "main"}
	if err := runNew(ctx); err != nil {
		t.Fatalf("new: %v", err)
	}

	if branch := gitRun(t, work, "rev-parse", "--abbrev-ref", "HEAD"); branch != "feature/login" {
		t.Errorf("branch atual = %q, esperado feature/login", branch)
	}
}

func TestNewExigeBranch(t *testing.T) {
	_, work := newSandbox(t)
	ctx := newContext(work)
	ctx.Opts = Options{Name: "new"}
	if err := runNew(ctx); err == nil {
		t.Error("esperado erro sem --branch")
	}
}

func TestCommitMessageAplicaSufixo(t *testing.T) {
	ctx := &Context{Project: config.Project{Name: "a", CommitSufixo: "teste-teste-teste"}}
	if got := ctx.commitMessage("feat: algo"); got != "feat: algo teste-teste-teste" {
		t.Errorf("commitMessage = %q", got)
	}
	semSufixo := &Context{Project: config.Project{Name: "a"}}
	if got := semSufixo.commitMessage("feat: algo"); got != "feat: algo" {
		t.Errorf("commitMessage = %q", got)
	}
}
