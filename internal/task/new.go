package task

import (
	"fmt"
	"strings"

	"git-manager/internal/ui"
)

func runNew(ctx *Context) error {
	branch := strings.TrimSpace(ctx.Opts.Branch)
	if branch == "" {
		return fmt.Errorf("informe a branch com --branch=nome-da-branch")
	}

	if _, err := update(ctx); err != nil {
		return err
	}

	if ctx.Git.BranchExists(branch) {
		ui.Warn("a branch %s já existe localmente, apenas fazendo checkout", branch)
		if err := ctx.Git.Checkout(branch); err != nil {
			return err
		}
		ctx.note("checkout na branch existente %s", branch)
		return nil
	}

	ui.Step("criando a branch %s", branch)
	if err := ctx.Git.CheckoutNew(branch); err != nil {
		return err
	}
	ui.Success("branch %s criada", branch)
	ctx.note("branch %s criada", branch)
	return nil
}
