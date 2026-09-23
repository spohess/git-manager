package task

import (
	"fmt"
	"strings"

	"git-manager/internal/ui"
)

func runCheckout(ctx *Context) error {
	branch := strings.TrimSpace(ctx.Opts.Branch)
	if branch == "" {
		return fmt.Errorf("informe a branch com --branch=nome-da-branch")
	}

	if err := runUpdate(ctx); err != nil {
		return err
	}

	repo := ctx.Git
	source := ctx.Opts.Source
	if branch == source {
		ui.Success("a branch %s é a de origem e já está atualizada", branch)
		ctx.note("branch %s atualizada", branch)
		return nil
	}

	switch {
	case repo.BranchExists(branch):
		ui.Step("checkout na branch local %s", branch)
		if err := repo.Checkout(branch); err != nil {
			return err
		}
	case repo.RemoteBranchExists(branch):
		ui.Step("checkout na branch %s/%s", repo.Remote, branch)
		if err := repo.CheckoutTracking(branch); err != nil {
			return err
		}
	default:
		return fmt.Errorf("a branch %s não existe localmente nem em %s; use a tarefa new para criá-la", branch, repo.Remote)
	}

	ui.Step("merge de %s em %s", source, branch)
	if err := repo.Merge(source); err != nil {
		ctx.note("checkout na branch %s (merge de %s em conflito, resolva manualmente)", branch, source)
		return fmt.Errorf("merge de %s em %s ficou com conflitos, resolva manualmente e finalize o commit: %w", source, branch, err)
	}

	ui.Success("branch %s em uso e atualizada com %s", branch, source)
	ctx.note("checkout na branch %s (atualizada com %s)", branch, source)
	return nil
}
