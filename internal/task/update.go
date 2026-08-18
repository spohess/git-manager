package task

import (
	"fmt"

	"git-manager/internal/git"
	"git-manager/internal/ui"
)

const autoCommitMessage = "processo automático"

func runUpdate(ctx *Context) error {
	_, err := update(ctx)
	return err
}

func update(ctx *Context) (string, error) {
	repo := ctx.Git

	if err := repo.Fetch(); err != nil {
		ui.Warn("fetch falhou, seguindo com as referências locais: %v", err)
	}

	mainBranch := repo.DefaultBranch()
	current, err := repo.CurrentBranch()
	if err != nil {
		return mainBranch, fmt.Errorf("não foi possível identificar a branch atual: %w", err)
	}
	ui.Step("branch atual: %s | branch principal: %s", current, mainBranch)

	if err := preserveLocalWork(ctx, current, mainBranch); err != nil {
		return mainBranch, err
	}

	if current != mainBranch {
		ui.Step("checkout para %s", mainBranch)
		if err := repo.Checkout(mainBranch); err != nil {
			ui.Warn("checkout bloqueado (%v), forçando", err)
			if err := repo.ForceCheckout(mainBranch); err != nil {
				return mainBranch, err
			}
		}
	}

	if err := syncMain(ctx, mainBranch); err != nil {
		return mainBranch, err
	}

	ui.Success("%s atualizada", mainBranch)
	if ctx.didReset {
		ctx.note("branch %s atualizada via reset --hard", mainBranch)
	} else {
		ctx.note("branch %s atualizada", mainBranch)
	}
	return mainBranch, nil
}

func preserveLocalWork(ctx *Context, current, mainBranch string) error {
	repo := ctx.Git
	dirty, err := repo.IsDirty()
	if err != nil {
		return err
	}
	if !dirty {
		return nil
	}
	if current == mainBranch {
		return nil
	}
	if current == git.DetachedHead {
		ui.Warn("detached HEAD com alterações locais: elas serão descartadas")
		return nil
	}

	ui.Step("alterações não commitadas em %s: commit automático e force push", current)
	if err := repo.StageAll(); err != nil {
		return err
	}
	if err := repo.Commit(autoCommitMessage); err != nil {
		return err
	}
	if err := repo.ForcePush(current); err != nil {
		return fmt.Errorf("force push em %s: %w", current, err)
	}
	return nil
}

func syncMain(ctx *Context, mainBranch string) error {
	repo := ctx.Git

	dirty, err := repo.IsDirty()
	if err != nil {
		return err
	}
	if dirty {
		ui.Warn("%s possui alterações locais, executando reset --hard", mainBranch)
		return resetToRemote(ctx, mainBranch)
	}

	ui.Step("pull %s %s", repo.Remote, mainBranch)
	if err := repo.Pull(mainBranch); err != nil {
		ui.Warn("pull bloqueado (%v), executando reset --hard", err)
		return resetToRemote(ctx, mainBranch)
	}
	return nil
}

func resetToRemote(ctx *Context, mainBranch string) error {
	repo := ctx.Git
	if !repo.RemoteBranchExists(mainBranch) {
		if err := repo.Fetch(); err != nil {
			return fmt.Errorf("reset em %s/%s: %w", repo.Remote, mainBranch, err)
		}
	}
	if err := repo.ResetHardRemote(mainBranch); err != nil {
		return err
	}
	ctx.didReset = true
	return nil
}
