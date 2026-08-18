# git-manager

CLI único em Go para operar o git em vários projetos ao mesmo tempo: atualizar as
branches principais, criar branches novas, alternar entre branches, abrir PRs com
mensagem gerada pelo `claude`, revisar e corrigir os apontamentos do PR.

## Instalação

```bash
make build           # gera ./bin/git-manager
make install         # instala em ~/.local/bin
```

Requisitos: Go 1.26+, `git`, e — para as tarefas `pr`, `review` e `fix` — o
`claude` no PATH. As tarefas `pr`, `draft` e `ready` também usam o `gh` (GitHub CLI)
autenticado.

## Configuração

Arquivo `.yml` com a lista de projetos:

```yaml
projects:
  - name: backend
    path: /Users/sergio/projects/backend
    main: true
    commit-sufixo: teste-teste-teste

  - name: admin
    path: /Users/sergio/projects/frontend-admin
    main: true

  - name: client
    path: /Users/sergio/projects/frontend-client
    main: false
```

| chave | obrigatória | descrição |
| --- | --- | --- |
| `name` | sim | nome usado em `--project=nome` |
| `path` | sim | diretório do repositório (aceita `~` e variáveis de ambiente) |
| `main` | sim | define se o projeto entra nas execuções sem `--no-main` |
| `commit-sufixo` | não | string acrescentada ao fim da mensagem de commit da tarefa `pr` |

Ordem de busca do arquivo quando `--config` não é informado:

1. variável de ambiente `GIT_MANAGER_CONFIG`;
2. diretório atual: `git-manager.yml`, `git-manager.yaml`, `projects.yml`,
   `projects.yaml`, `config.yml`, `config.yaml`;
3. `~/.config/git-manager/config.yml` (e `.yaml`);
4. `~/.git-manager.yml` (e `.yaml`).

## Uso

```bash
git-manager <tarefa> [parâmetros]
```

### Parâmetros globais

| parâmetro | efeito |
| --- | --- |
| `--no-main` | executa em todos os projetos, inclusive os com `main: false` |
| `--project=nome` | executa apenas no(s) projeto(s) informado(s) (ignora o filtro de `main`) |
| `--branch=nome` | nome da branch (obrigatório nas tarefas `new` e `checkout`; opcional nas `draft` e `ready`) |
| `--config=arquivo.yml` | caminho do arquivo de configuração |
| `--dry-run` | imprime os comandos sem aplicar nenhuma alteração |

Sem `--no-main` e sem `--project`, a tarefa roda somente nos projetos com
`main: true`.

`--project` aceita múltiplos nomes separados por vírgula (ex.:
`--project=backend,admin`), executando a tarefa em cada um deles, na ordem
informada. Nomes repetidos são ignorados e um nome inexistente falha a
execução inteira.

### Tarefas

#### `update`

1. `git fetch --prune origin`;
2. se a branch atual tiver alterações não commitadas (e não for a principal):
   `git add --all`, `git commit --no-verify -m "processo automático"` e
   `git push --force origin <branch>`;
3. `git checkout <principal>` (com `--force` caso o checkout normal seja
   bloqueado);
4. `git pull origin <principal>`; se a principal tiver alterações locais ou o
   pull for bloqueado, aplica `git reset --hard origin/<principal>`.

#### `new --branch=nome`

Executa o `update` e em seguida `git checkout -b nome`. Se a branch já existir
localmente, apenas faz o checkout nela e avisa.

```bash
git-manager new --branch=feature/login --no-main
```

#### `checkout --branch=nome`

Executa o `update` — inclusive o commit automático com a mensagem
`processo automático` e o force push da branch atual — e em seguida entra na
branch informada:

1. se ela existir localmente: `git checkout <nome>`;
2. senão, se existir em `origin`: `git checkout -b <nome> --track origin/<nome>`;
3. se não existir em nenhum dos dois, a tarefa falha e indica o uso do `new`.

Informar a própria branch principal é aceito: o `update` já deixa o repositório
nela, atualizada.

```bash
git-manager checkout --branch=feature/login --no-main
```

#### `pr`

1. `git add --all`;
2. gera título e descrição com
   `claude -p "@pr-message gere a mensagem das alterações no formato markdown" --allowedTools "Read,Edit,Bash,Git"`;
3. `git commit --no-verify -m "<title> <commit-sufixo>"`;
4. `git push --set-upstream origin <branch>` (com fallback para `--force`);
5. `gh pr create --draft --assignee @me` usando o `title` como título e o
   `message` como descrição — o PR é sempre aberto como draft e atribuído ao
   usuário autenticado no `gh`.

Chamadas ao `gh` que falham por indisponibilidade do GitHub (HTTP 5xx, timeout)
são repetidas até três vezes. Se ainda assim a criação com assignee falhar, o PR
é criado sem assignee e a atribuição é tentada em seguida com
`gh pr edit --add-assignee @me`; persistindo o erro, um aviso pede a atribuição
manual.

Se já existir um PR aberto para a branch, o push atualiza o PR existente, a URL
é exibida no resumo e, se ele não tiver assignee, é atribuído a você. A tarefa
falha se a branch atual for a principal ou se não houver diferença em relação a
ela.

A leitura da resposta do `claude` aceita três formatos, nesta ordem: as seções
`title:`/`message:` (com ou sem blocos ```); dois blocos ``` sem rótulo, sendo o
primeiro o título e o segundo a descrição — formato comum no modo `-p`, que
costuma perder os rótulos e acrescentar um preâmbulo; ou um único bloco ```, em
que a primeira linha vira o título e o restante a descrição. Texto sem nenhum
bloco é recusado com erro, e a saída bruta é impressa para conferência.

#### `draft`

Converte para draft o PR da branch atual, usando `gh pr ready --undo`. Com
`--branch=nome` o PR daquela branch é usado sem trocar de branch no repositório.

- se não houver PR aberto para a branch, a tarefa falha;
- se o PR já estiver em draft, nada é alterado e o resumo informa isso.

```bash
git-manager draft --no-main
git-manager draft --branch=SYSVET-924 --project=backend
```

#### `ready`

Marca como pronto para revisão o PR da branch atual, usando `gh pr ready`. Com
`--branch=nome` o PR daquela branch é usado sem trocar de branch no repositório.

- se não houver PR aberto para a branch, a tarefa falha;
- se o PR já estiver pronto para revisão, nada é alterado e o resumo informa isso.

```bash
git-manager ready --no-main
git-manager ready --branch=SYSVET-924 --project=backend
```

#### `review`

`claude -p "@pr-reviewer revise o pr da branch atual" --allowedTools "Read,Edit,Bash,Git"`,
com a saída transmitida em tempo real.

#### `fix`

`claude -p "@pr-comment-fixer corrija os apontamentos feitos no pr a branch corrente mesmo que forem problemas pré-existentes, pode corrigir" --allowedTools "Read,Edit,Bash,Git"`.

#### `prune`

1. `git fetch --prune origin`;
2. lista as branches locais cujo upstream foi removido (`[gone]` no
   `git for-each-ref --format='%(upstream:track)'`);
3. remove cada uma com `git branch -D`, pulando a branch atualmente
   selecionada (com aviso) caso ela esteja nessa lista.

```bash
git-manager prune --no-main
```

## Exemplos

```bash
git-manager update                                  # projetos main: true
git-manager update --no-main                        # todos os projetos
git-manager new --branch=feature/login --no-main
git-manager checkout --branch=feature/login --no-main
git-manager pr --project=backend
git-manager draft --no-main                          # PR da branch atual volta a draft
git-manager ready --no-main                          # PR da branch atual sai do draft
git-manager review --no-main
git-manager fix --project=backend
git-manager prune --project=backend,admin           # múltiplos projetos
git-manager update --no-main --dry-run              # simula, sem alterar nada
```

## Detalhes de implementação

- todos os commits usam `--no-verify`;
- a branch principal é detectada por `origin/HEAD` e, na ausência dela, por
  `main` e depois `master`;
- o `commit-sufixo` é aplicado no commit da tarefa `pr`; o commit automático do
  `update` mantém a mensagem fixa `processo automático`;
- quando a branch atual **é** a principal e existem alterações locais, elas são
  descartadas pelo `reset --hard origin/<principal>` (regra 5.1.b), em vez de
  gerarem um force push na principal;
- os projetos são processados em sequência e o resumo final lista o resultado de
  cada um; o processo sai com código 1 se algum projeto falhar.

## Desenvolvimento

```bash
make fmt
make vet
make test
```
