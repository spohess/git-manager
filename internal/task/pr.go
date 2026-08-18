package task

import (
	"fmt"
	"strings"

	"git-manager/internal/claude"
	"git-manager/internal/gh"
	"git-manager/internal/git"
	"git-manager/internal/ui"
)

const dryRunTitle = "título gerado pelo claude"

func runPR(ctx *Context) error {
	repo := ctx.Git

	branch, err := repo.CurrentBranch()
	if err != nil {
		return fmt.Errorf("não foi possível identificar a branch atual: %w", err)
	}
	if branch == git.DetachedHead {
		return fmt.Errorf("o repositório está em detached HEAD; faça checkout em uma branch")
	}
	mainBranch := repo.DefaultBranch()
	if branch == mainBranch {
		return fmt.Errorf("a branch atual é a principal (%s); crie uma branch com a tarefa new", mainBranch)
	}

	dirty, err := repo.IsDirty()
	if err != nil {
		return err
	}
	if !dirty && !repo.HasCommitsAhead(mainBranch, branch) {
		return fmt.Errorf("nenhuma alteração entre %s/%s e %s", repo.Remote, mainBranch, branch)
	}

	if dirty {
		if err := repo.StageAll(); err != nil {
			return err
		}
	}

	message, err := generateMessage(ctx)
	if err != nil {
		return err
	}
	ui.Step("título: %s", message.Title)

	if dirty {
		if err := repo.Commit(ctx.commitMessage(message.Title)); err != nil {
			return err
		}
	} else {
		ui.Info("nada para commitar, usando os commits já existentes na branch")
	}

	ui.Step("push da branch %s", branch)
	if err := repo.Push(branch); err != nil {
		ui.Warn("push rejeitado (%v), tentando force push", err)
		if err := repo.ForcePush(branch); err != nil {
			return err
		}
	}

	return openPullRequest(ctx, branch, mainBranch, message)
}

func generateMessage(ctx *Context) (claude.Message, error) {
	if ctx.Opts.DryRun {
		ui.Skipped(claude.Describe(claude.PromptMessage))
		return claude.Message{Title: dryRunTitle, Body: "descrição gerada pelo claude"}, nil
	}
	ui.Step("gerando a mensagem com o claude (pode demorar)")
	ui.Command(claude.Describe(claude.PromptMessage))
	raw, err := claude.Capture(ctx.Project.Path, claude.PromptMessage)
	if err != nil {
		return claude.Message{}, err
	}
	message, err := claude.ParseMessage(raw)
	if err != nil {
		ui.Output(strings.TrimSpace(raw))
		return claude.Message{}, err
	}
	return message, nil
}

func openPullRequest(ctx *Context, branch, mainBranch string, message claude.Message) error {
	existing, err := gh.Current(ctx.Project.Path)
	if err != nil {
		return err
	}
	if existing != nil {
		ui.Success("PR #%d já existe e foi atualizado com o push: %s", existing.Number, existing.URL)
		if !existing.HasAssignee() {
			if err := gh.AssignSelf(ctx.Project.Path, ctx.Opts.DryRun); err != nil {
				ui.Warn("não foi possível atribuir o PR #%d a você: %v", existing.Number, err)
			}
		}
		ctx.note("PR #%d atualizado", existing.Number)
		return nil
	}

	ui.Step("criando o PR em draft contra %s", mainBranch)
	url, err := gh.Create(ctx.Project.Path, message.Title, message.Body, mainBranch, branch, ctx.Opts.DryRun)
	if err != nil {
		return err
	}
	if ctx.Opts.DryRun {
		ctx.note("PR draft seria criado a partir de %s", branch)
		return nil
	}
	ui.Success("PR draft criado: %s", url)
	ctx.note("PR draft criado: %s", url)
	return nil
}
