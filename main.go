package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"

	"git-manager/internal/config"
	"git-manager/internal/task"
	"git-manager/internal/ui"
)

var version = "dev"

func main() {
	if err := run(os.Args[1:]); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			os.Exit(2)
		}
		ui.Fail("%v", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	name, rest := splitCommand(args)

	switch name {
	case "", "help":
		usage(os.Stdout)
		return nil
	case "version":
		fmt.Printf("git-manager %s\n", version)
		return nil
	}

	if !task.Exists(name) {
		return fmt.Errorf("tarefa desconhecida: %q (disponíveis: %s)", name, strings.Join(task.Names(), ", "))
	}

	fs := flag.NewFlagSet("git-manager "+name, flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	fs.Usage = func() { usage(os.Stderr) }
	noMain := fs.Bool("no-main", false, "executa em todos os projetos, inclusive os com main: false")
	project := fs.String("project", "", "executa apenas no(s) projeto(s) informado(s), separados por vírgula")
	branch := fs.String("branch", "", "nome da branch (obrigatório nas tarefas new e checkout)")
	configPath := fs.String("config", "", "caminho do arquivo .yml de configuração")
	dryRun := fs.Bool("dry-run", false, "mostra os comandos sem executar as alterações")
	if err := fs.Parse(rest); err != nil {
		return err
	}
	if fs.NArg() > 0 {
		return fmt.Errorf("argumento inesperado: %q", fs.Arg(0))
	}

	cfg, path, err := config.Load(*configPath)
	if err != nil {
		return err
	}
	ui.Info("configuração: %s", path)

	return task.Run(cfg, task.Options{
		Name:    name,
		Branch:  *branch,
		NoMain:  *noMain,
		Project: *project,
		DryRun:  *dryRun,
	})
}

var valueFlags = map[string]bool{
	"project": true,
	"branch":  true,
	"config":  true,
}

func splitCommand(args []string) (string, []string) {
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if !strings.HasPrefix(arg, "-") {
			return arg, append(append([]string{}, args[:i]...), args[i+1:]...)
		}
		normalized := strings.TrimLeft(arg, "-")
		if strings.Contains(normalized, "=") {
			continue
		}
		if valueFlags[normalized] {
			i++
		}
	}
	if len(args) == 1 {
		switch args[0] {
		case "-h", "--help":
			return "help", nil
		case "-v", "--version":
			return "version", nil
		}
	}
	return "", args
}

func usage(out *os.File) {
	fmt.Fprintf(out, `git-manager %s - gerenciador git para múltiplos projetos

Uso:
  git-manager <tarefa> [parâmetros]

Tarefas:
`, version)
	for _, line := range task.Summaries() {
		fmt.Fprintf(out, "  %s\n", line)
	}
	fmt.Fprint(out, `
Parâmetros:
  --no-main            executa em todos os projetos, inclusive os com main: false
  --project=nome       executa apenas no(s) projeto(s) informado(s) (ignora o filtro
                        main); aceita múltiplos nomes separados por vírgula
  --branch=nome        nome da branch (obrigatório nas tarefas new e checkout,
                        opcional nas draft e ready)
  --config=arquivo.yml caminho do arquivo de configuração
  --dry-run            mostra os comandos sem aplicar alterações

Configuração:
  Sem --config o arquivo é procurado em GIT_MANAGER_CONFIG, no diretório atual
  (git-manager.yml, projects.yml, config.yml) e em ~/.config/git-manager/config.yml.

Exemplos:
  git-manager status
  git-manager update
  git-manager update --no-main
  git-manager new --branch=feature/login --project=backend
  git-manager checkout --branch=feature/login --no-main
  git-manager pr --no-main
  git-manager draft --branch=feature/login --project=backend
  git-manager ready --branch=feature/login --project=backend
  git-manager review --project=backend
  git-manager fix
  git-manager prune --project=backend,admin
`)
}
