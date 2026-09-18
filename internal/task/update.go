package task

import (
	"fmt"

	"git-manager/internal/git"
	"git-manager/internal/ui"
)

const autoCommitMessage = "processo automático"

func runUpdate(ctx *Context) error {
	repo := ctx.Git
	target := ctx.Opts.Target

	if err := repo.Fetch(); err != nil {
		ui.Warn("fetch falhou, seguindo com as referências locais: %v", err)
	}
	if !repo.RemoteBranchExists(target) {
		return fmt.Errorf("a branch de destino %s não existe em %s; confira o branch-target do config ou o --target", target, repo.Remote)
	}

	current, err := repo.CurrentBranch()
	if err != nil {
		return fmt.Errorf("não foi possível identificar a branch atual: %w", err)
	}
	ui.Step("branch atual: %s | branch de destino: %s", current, target)

	if err := preserveLocalWork(ctx, current, target); err != nil {
		return err
	}

	if current != target {
		ui.Step("checkout para %s", target)
		if err := repo.Checkout(target); err != nil {
			ui.Warn("checkout bloqueado (%v), forçando", err)
			if err := repo.ForceCheckout(target); err != nil {
				return err
			}
		}
	}

	if err := syncTarget(ctx, target); err != nil {
		return err
	}

	ui.Success("%s atualizada", target)
	if ctx.didReset {
		ctx.note("branch %s atualizada via reset --hard", target)
	} else {
		ctx.note("branch %s atualizada", target)
	}
	return nil
}

func preserveLocalWork(ctx *Context, current, target string) error {
	repo := ctx.Git
	dirty, err := repo.IsDirty()
	if err != nil {
		return err
	}
	if !dirty {
		return nil
	}
	if current == target {
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

func syncTarget(ctx *Context, target string) error {
	repo := ctx.Git

	dirty, err := repo.IsDirty()
	if err != nil {
		return err
	}
	if dirty {
		ui.Warn("%s possui alterações locais, executando reset --hard", target)
		return resetToRemote(ctx, target)
	}

	ui.Step("pull %s %s", repo.Remote, target)
	if err := repo.Pull(target); err != nil {
		ui.Warn("pull bloqueado (%v), executando reset --hard", err)
		return resetToRemote(ctx, target)
	}
	return nil
}

func resetToRemote(ctx *Context, target string) error {
	repo := ctx.Git
	if !repo.RemoteBranchExists(target) {
		if err := repo.Fetch(); err != nil {
			return fmt.Errorf("reset em %s/%s: %w", repo.Remote, target, err)
		}
	}
	if err := repo.ResetHardRemote(target); err != nil {
		return err
	}
	ctx.didReset = true
	return nil
}
