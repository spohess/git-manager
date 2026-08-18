package task

import "testing"

func TestPruneRemoveBranchComUpstreamApagado(t *testing.T) {
	_, work := newSandbox(t)

	gitRun(t, work, "checkout", "-b", "feature/x")
	gitRun(t, work, "push", "--set-upstream", "origin", "feature/x")
	gitRun(t, work, "checkout", "main")
	gitRun(t, work, "push", "origin", "--delete", "feature/x")

	ctx := newContext(work)
	ctx.Opts = Options{Name: "prune"}
	if err := runPrune(ctx); err != nil {
		t.Fatalf("prune: %v", err)
	}

	if ctx.Git.BranchExists("feature/x") {
		t.Error("feature/x deveria ter sido removida")
	}
}

func TestPrunePreservaBranchAtualComUpstreamApagado(t *testing.T) {
	_, work := newSandbox(t)

	gitRun(t, work, "checkout", "-b", "feature/y")
	gitRun(t, work, "push", "--set-upstream", "origin", "feature/y")
	gitRun(t, work, "push", "origin", "--delete", "feature/y")

	ctx := newContext(work)
	ctx.Opts = Options{Name: "prune"}
	if err := runPrune(ctx); err != nil {
		t.Fatalf("prune: %v", err)
	}

	if !ctx.Git.BranchExists("feature/y") {
		t.Error("a branch atual não deveria ser removida, mesmo com upstream apagado")
	}
}

func TestPruneSemBranchesParaRemover(t *testing.T) {
	_, work := newSandbox(t)

	ctx := newContext(work)
	ctx.Opts = Options{Name: "prune"}
	if err := runPrune(ctx); err != nil {
		t.Fatalf("prune: %v", err)
	}
}
