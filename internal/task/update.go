package task

import (
	"fmt"

	"git-manager/internal/git"
	"git-manager/internal/ui"
)

const autoCommitMessage = "processo automático"

func runUpdate(ctx *Context) error {
	repo := ctx.Git
	source := ctx.Opts.Source

	if err := repo.Fetch(); err != nil {
		ui.Warn("fetch falhou, seguindo com as referências locais: %v", err)
	}
	if !repo.RemoteBranchExists(source) {
		return fmt.Errorf("a branch de origem %s não existe em %s; confira o major-branch do config ou o --source", source, repo.Remote)
	}

	current, err := repo.CurrentBranch()
	if err != nil {
		return fmt.Errorf("não foi possível identificar a branch atual: %w", err)
	}
	ui.Step("branch atual: %s | branch de origem: %s", current, source)

	if err := preserveLocalWork(ctx, current, source); err != nil {
		return err
	}

	if current != source {
		ui.Step("checkout para %s", source)
		if err := repo.Checkout(source); err != nil {
			ui.Warn("checkout bloqueado (%v), forçando", err)
			if err := repo.ForceCheckout(source); err != nil {
				return err
			}
		}
	}

	if err := syncSource(ctx, source); err != nil {
		return err
	}

	ui.Success("%s atualizada", source)
	if ctx.didReset {
		ctx.note("branch %s atualizada via reset --hard", source)
	} else {
		ctx.note("branch %s atualizada", source)
	}
	return nil
}

func preserveLocalWork(ctx *Context, current, source string) error {
	repo := ctx.Git
	dirty, err := repo.IsDirty()
	if err != nil {
		return err
	}
	if !dirty {
		return nil
	}
	if current == source {
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

func syncSource(ctx *Context, source string) error {
	repo := ctx.Git

	dirty, err := repo.IsDirty()
	if err != nil {
		return err
	}
	if dirty {
		ui.Warn("%s possui alterações locais, executando reset --hard", source)
		return resetToRemote(ctx, source)
	}

	ui.Step("pull %s %s", repo.Remote, source)
	if err := repo.Pull(source); err != nil {
		ui.Warn("pull bloqueado (%v), executando reset --hard", err)
		return resetToRemote(ctx, source)
	}
	return nil
}

func resetToRemote(ctx *Context, source string) error {
	repo := ctx.Git
	if !repo.RemoteBranchExists(source) {
		if err := repo.Fetch(); err != nil {
			return fmt.Errorf("reset em %s/%s: %w", repo.Remote, source, err)
		}
	}
	if err := repo.ResetHardRemote(source); err != nil {
		return err
	}
	ctx.didReset = true
	return nil
}
