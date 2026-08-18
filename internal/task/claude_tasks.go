package task

import (
	"fmt"

	"git-manager/internal/claude"
	"git-manager/internal/git"
	"git-manager/internal/ui"
)

func runReview(ctx *Context) error {
	return runClaudeOnBranch(ctx, claude.PromptReview, "review", "review executado")
}

func runFix(ctx *Context) error {
	return runClaudeOnBranch(ctx, claude.PromptFix, "fix", "fix executado")
}

func runClaudeOnBranch(ctx *Context, prompt, label, done string) error {
	branch, err := ctx.Git.CurrentBranch()
	if err != nil {
		return fmt.Errorf("não foi possível identificar a branch atual: %w", err)
	}
	if branch == git.DetachedHead {
		return fmt.Errorf("o repositório está em detached HEAD; faça checkout em uma branch")
	}
	ui.Step("%s na branch %s", label, branch)
	if ctx.Opts.DryRun {
		ui.Skipped(claude.Describe(prompt))
		ctx.note("%s seria executado em %s", label, branch)
		return nil
	}
	ui.Command(claude.Describe(prompt))
	if err := claude.Stream(ctx.Project.Path, prompt); err != nil {
		return err
	}
	ui.Success("%s", done)
	ctx.note("%s em %s", done, branch)
	return nil
}
