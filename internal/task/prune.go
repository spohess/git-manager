package task

import (
	"fmt"

	"git-manager/internal/ui"
)

func runPrune(ctx *Context) error {
	repo := ctx.Git

	if err := repo.Fetch(); err != nil {
		return err
	}

	current, err := repo.CurrentBranch()
	if err != nil {
		return fmt.Errorf("não foi possível identificar a branch atual: %w", err)
	}

	gone, err := repo.GoneBranches()
	if err != nil {
		return fmt.Errorf("não foi possível listar as branches locais: %w", err)
	}

	deleted := 0
	for _, branch := range gone {
		if branch == current {
			ui.Warn("branch atual %s está com upstream removido, pulando", branch)
			continue
		}
		ui.Step("removendo %s (upstream removido)", branch)
		if err := repo.DeleteBranch(branch); err != nil {
			return err
		}
		deleted++
	}

	if deleted == 0 {
		ui.Success("nenhuma branch com upstream removido")
		ctx.note("nenhuma branch removida")
		return nil
	}

	ui.Success("%d branch(es) removida(s)", deleted)
	ctx.note("%d branch(es) removida(s)", deleted)
	return nil
}
