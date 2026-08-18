package task

import (
	"fmt"
	"strings"

	"git-manager/internal/gh"
	"git-manager/internal/git"
	"git-manager/internal/ui"
)

func runDraft(ctx *Context) error {
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
	if pr.IsDraft {
		ui.Success("PR #%d já está em draft: %s", pr.Number, pr.URL)
		ctx.note("PR #%d já estava em draft", pr.Number)
		return nil
	}

	ui.Step("convertendo o PR #%d da branch %s para draft", pr.Number, branch)
	if err := gh.MarkDraft(ctx.Project.Path, branch, ctx.Opts.DryRun); err != nil {
		return err
	}
	if ctx.Opts.DryRun {
		ctx.note("PR #%d seria convertido para draft", pr.Number)
		return nil
	}
	ui.Success("PR #%d convertido para draft: %s", pr.Number, pr.URL)
	ctx.note("PR #%d convertido para draft", pr.Number)
	return nil
}

func targetBranch(ctx *Context) (string, error) {
	if branch := strings.TrimSpace(ctx.Opts.Branch); branch != "" {
		return branch, nil
	}
	branch, err := ctx.Git.CurrentBranch()
	if err != nil {
		return "", fmt.Errorf("não foi possível identificar a branch atual: %w", err)
	}
	if branch == git.DetachedHead {
		return "", fmt.Errorf("o repositório está em detached HEAD; informe --branch=nome ou faça checkout em uma branch")
	}
	return branch, nil
}
