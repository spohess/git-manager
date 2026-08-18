package task

import (
	"fmt"

	"git-manager/internal/gh"
	"git-manager/internal/ui"
)

func runReady(ctx *Context) error {
	branch, err := targetBranch(ctx)
	if err != nil {
		return err
	}

	pr, err := gh.View(ctx.Project.Path, branch)
	if err != nil {
		return err
	}
	if pr == nil {
		return fmt.Errorf("nenhum PR aberto para a branch %s", branch)
	}
	if !pr.IsDraft {
		ui.Success("PR #%d já está pronto para revisão: %s", pr.Number, pr.URL)
		ctx.note("PR #%d já estava pronto para revisão", pr.Number)
		return nil
	}

	ui.Step("marcando o PR #%d da branch %s como pronto para revisão", pr.Number, branch)
	if err := gh.MarkReady(ctx.Project.Path, branch, ctx.Opts.DryRun); err != nil {
		return err
	}
	if ctx.Opts.DryRun {
		ctx.note("PR #%d seria marcado como pronto para revisão", pr.Number)
		return nil
	}
	ui.Success("PR #%d marcado como pronto para revisão: %s", pr.Number, pr.URL)
	ctx.note("PR #%d marcado como pronto para revisão", pr.Number)
	return nil
}
