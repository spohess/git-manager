package task

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"git-manager/internal/claude"
	"git-manager/internal/config"
	"git-manager/internal/gh"
	"git-manager/internal/git"
	"git-manager/internal/ui"
)

type Options struct {
	Name    string
	Branch  string
	NoMain  bool
	Project string
	DryRun  bool
}

type Context struct {
	Project config.Project
	Git     *git.Client
	Opts    Options

	detail   string
	didReset bool
}

type definition struct {
	run          func(*Context) error
	summary      string
	needsBranch  bool
	needsClaude  bool
	needsGh      bool
	needsRemote  bool
	requiresRepo bool
}

var definitions = map[string]definition{
	"update":   {run: runUpdate, summary: "checkout na branch principal e pull, preservando o trabalho local", needsRemote: true, requiresRepo: true},
	"new":      {run: runNew, summary: "executa o update e cria uma nova branch (--branch=nome)", needsBranch: true, needsRemote: true, requiresRepo: true},
	"checkout": {run: runCheckout, summary: "executa o update e faz checkout na branch informada (--branch=nome)", needsBranch: true, needsRemote: true, requiresRepo: true},
	"prune":    {run: runPrune, summary: "fetch --prune e remove as branches locais cujo upstream foi apagado", needsRemote: true, requiresRepo: true},
	"draft":    {run: runDraft, summary: "converte o PR da branch atual (ou --branch=nome) para draft", needsGh: true, needsRemote: true, requiresRepo: true},
	"ready":    {run: runReady, summary: "marca o PR da branch atual (ou --branch=nome) como pronto para revisão", needsGh: true, needsRemote: true, requiresRepo: true},
	"pr":       {run: runPR, summary: "gera a mensagem com o claude, commita, faz push e abre o PR", needsClaude: true, needsGh: true, needsRemote: true, requiresRepo: true},
	"review":   {run: runReview, summary: "revisa o PR da branch atual com o claude", needsClaude: true, requiresRepo: true},
	"fix":      {run: runFix, summary: "corrige os apontamentos do PR da branch atual com o claude", needsClaude: true, requiresRepo: true},
}

func Exists(name string) bool {
	_, ok := definitions[name]
	return ok
}

func Names() []string {
	list := make([]string, 0, len(definitions))
	for name := range definitions {
		list = append(list, name)
	}
	sort.Strings(list)
	return list
}

func Summaries() []string {
	list := make([]string, 0, len(definitions))
	for _, name := range Names() {
		list = append(list, fmt.Sprintf("%-8s %s", name, definitions[name].summary))
	}
	return list
}

type result struct {
	name   string
	detail string
	err    error
}

func Run(cfg *config.Config, opts Options) error {
	def, ok := definitions[opts.Name]
	if !ok {
		return fmt.Errorf("tarefa desconhecida: %q", opts.Name)
	}
	if def.needsBranch && strings.TrimSpace(opts.Branch) == "" {
		return fmt.Errorf("a tarefa %s exige o parâmetro --branch=nome-da-branch", opts.Name)
	}
	if err := git.Available(); err != nil {
		return err
	}
	if def.needsClaude {
		if err := claude.Available(); err != nil {
			return err
		}
	}
	if def.needsGh {
		if err := gh.Available(); err != nil {
			return err
		}
	}
	projects, err := config.Select(cfg, opts.Project, opts.NoMain)
	if err != nil {
		return err
	}

	ui.Info("tarefa: %s | projetos: %d%s", opts.Name, len(projects), dryRunLabel(opts.DryRun))

	results := make([]result, 0, len(projects))
	for _, project := range projects {
		ui.Project(project.Name, project.Path)
		ctx := &Context{
			Project: project,
			Git:     git.New(project.Path, opts.DryRun),
			Opts:    opts,
		}
		err := execute(ctx, def)
		if err != nil {
			ui.Fail("%s: %v", project.Name, err)
		}
		results = append(results, result{name: project.Name, detail: ctx.detail, err: err})
	}
	return report(results)
}

func dryRunLabel(dryRun bool) string {
	if dryRun {
		return " | modo dry-run (nenhuma alteração será aplicada)"
	}
	return ""
}

func execute(ctx *Context, def definition) error {
	info, err := os.Stat(ctx.Project.Path)
	if err != nil || !info.IsDir() {
		return fmt.Errorf("diretório inexistente: %s", ctx.Project.Path)
	}
	if def.requiresRepo && !ctx.Git.IsRepo() {
		return fmt.Errorf("%s não é um repositório git", ctx.Project.Path)
	}
	if def.needsRemote && !ctx.Git.HasRemote() {
		return fmt.Errorf("o repositório não possui o remote %q", ctx.Git.Remote)
	}
	return def.run(ctx)
}

func report(results []result) error {
	ui.Section("Resumo")
	failures := 0
	for _, r := range results {
		if r.err != nil {
			failures++
			ui.StatusFail(r.name, r.err.Error())
			continue
		}
		ui.StatusOK(r.name, r.detail)
	}
	if failures > 0 {
		return fmt.Errorf("%d de %d projeto(s) falharam", failures, len(results))
	}
	return nil
}

func (c *Context) note(format string, args ...any) {
	c.detail = fmt.Sprintf(format, args...)
}

func (c *Context) commitMessage(title string) string {
	if c.Project.CommitSufixo == "" {
		return title
	}
	return strings.TrimSpace(title) + " " + c.Project.CommitSufixo
}
