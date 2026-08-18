package task

import "testing"

func checkoutContext(path, branch string) *Context {
	ctx := newContext(path)
	ctx.Opts = Options{Name: "checkout", Branch: branch}
	return ctx
}

func TestCheckoutBranchLocalAposUpdate(t *testing.T) {
	_, work := newSandbox(t)

	gitRun(t, work, "checkout", "-b", "feature/x")
	writeFile(t, work, "novo.txt", "trabalho em andamento\n")
	gitRun(t, work, "checkout", "main")

	gitRun(t, work, "checkout", "feature/x")
	if err := runCheckout(checkoutContext(work, "feature/x")); err != nil {
		t.Fatalf("checkout: %v", err)
	}

	if branch := gitRun(t, work, "rev-parse", "--abbrev-ref", "HEAD"); branch != "feature/x" {
		t.Errorf("branch atual = %q, esperado feature/x", branch)
	}
	if status := gitRun(t, work, "status", "--porcelain"); status != "" {
		t.Errorf("árvore suja após o checkout: %q", status)
	}
	if subject := gitRun(t, work, "log", "-1", "--format=%s", "origin/feature/x"); subject != autoCommitMessage {
		t.Errorf("mensagem do commit automático = %q, esperado %q", subject, autoCommitMessage)
	}
}

func TestCheckoutBranchRemotaCriaLocalComTracking(t *testing.T) {
	_, work := newSandbox(t)

	gitRun(t, work, "checkout", "-b", "feature/remota")
	writeFile(t, work, "remoto.txt", "conteúdo\n")
	gitRun(t, work, "add", ".")
	gitRun(t, work, "commit", "-m", "remoto")
	gitRun(t, work, "push", "--set-upstream", "origin", "feature/remota")
	gitRun(t, work, "checkout", "main")
	gitRun(t, work, "branch", "-D", "feature/remota")

	if err := runCheckout(checkoutContext(work, "feature/remota")); err != nil {
		t.Fatalf("checkout: %v", err)
	}

	if branch := gitRun(t, work, "rev-parse", "--abbrev-ref", "HEAD"); branch != "feature/remota" {
		t.Errorf("branch atual = %q, esperado feature/remota", branch)
	}
	if upstream := gitRun(t, work, "rev-parse", "--abbrev-ref", "@{upstream}"); upstream != "origin/feature/remota" {
		t.Errorf("upstream = %q, esperado origin/feature/remota", upstream)
	}
	if head := gitRun(t, work, "rev-parse", "HEAD"); head != gitRun(t, work, "rev-parse", "origin/feature/remota") {
		t.Error("a branch local não aponta para o mesmo commit da remota")
	}
}

func TestCheckoutBranchInexistenteFalha(t *testing.T) {
	_, work := newSandbox(t)

	if err := runCheckout(checkoutContext(work, "feature/nao-existe")); err == nil {
		t.Error("esperado erro para branch inexistente")
	}
	if branch := gitRun(t, work, "rev-parse", "--abbrev-ref", "HEAD"); branch != "main" {
		t.Errorf("branch atual = %q, esperado main", branch)
	}
}

func TestCheckoutNaBranchPrincipal(t *testing.T) {
	_, work := newSandbox(t)

	if err := runCheckout(checkoutContext(work, "main")); err != nil {
		t.Fatalf("checkout: %v", err)
	}
	if branch := gitRun(t, work, "rev-parse", "--abbrev-ref", "HEAD"); branch != "main" {
		t.Errorf("branch atual = %q, esperado main", branch)
	}
}

func TestCheckoutExigeBranch(t *testing.T) {
	_, work := newSandbox(t)

	if err := runCheckout(checkoutContext(work, "")); err == nil {
		t.Error("esperado erro sem --branch")
	}
}
