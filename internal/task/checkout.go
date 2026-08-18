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

	mainBranch, err := update(ctx)
	if err != nil {
		return err
	}

	repo := ctx.Git
	if branch == mainBranch {
		ui.Success("a branch %s é a principal e já está atualizada", branch)
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

	ui.Success("branch %s em uso", branch)
	ctx.note("checkout na branch %s", branch)
	return nil
}
