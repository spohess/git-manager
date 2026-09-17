package provider

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"
	"strconv"
	"strings"

	"git-manager/internal/ui"
)

const (
	selfAssignee     = "@me"
	githubViewFields = "number,url,state,title,isDraft,assignees"
)

type githubUser struct {
	Login string `json:"login"`
}

type githubPullRequest struct {
	Number    int          `json:"number"`
	URL       string       `json:"url"`
	State     string       `json:"state"`
	Title     string       `json:"title"`
	IsDraft   bool         `json:"isDraft"`
	Assignees []githubUser `json:"assignees"`
}

type GitHub struct {
	dir string
}

func NewGitHub(dir string) *GitHub {
	return &GitHub{dir: dir}
}

func githubAvailable() error {
	if _, err := exec.LookPath("gh"); err != nil {
		return fmt.Errorf("comando \"gh\" (GitHub CLI) não encontrado no PATH")
	}
	return nil
}

func (g *GitHub) run(stdin string, args ...string) (string, error) {
	cmd := exec.Command("gh", args...)
	cmd.Dir = g.dir
	if stdin != "" {
		cmd.Stdin = strings.NewReader(stdin)
	}
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		detail := strings.TrimSpace(stderr.String())
		if detail == "" {
			detail = err.Error()
		}
		return strings.TrimSpace(stdout.String()), fmt.Errorf("gh %s falhou: %s", strings.Join(args, " "), detail)
	}
	return strings.TrimSpace(stdout.String()), nil
}

func (g *GitHub) runRetry(stdin string, args ...string) (string, error) {
	var out string
	err := withRetry("github", func() error {
		var err error
		out, err = g.run(stdin, args...)
		return err
	})
	return out, err
}

func (g *GitHub) Find(branch string) (*PullRequest, error) {
	args := []string{"pr", "view"}
	if branch = strings.TrimSpace(branch); branch != "" {
		args = append(args, branch)
	}
	args = append(args, "--json", githubViewFields)
	out, err := g.run("", args...)
	if err != nil {
		return nil, nil
	}
	var pr githubPullRequest
	if err := json.Unmarshal([]byte(out), &pr); err != nil {
		return nil, fmt.Errorf("resposta inesperada do gh pr view: %w", err)
	}
	if pr.Number == 0 || strings.EqualFold(pr.State, "CLOSED") || strings.EqualFold(pr.State, "MERGED") {
		return nil, nil
	}
	return &PullRequest{
		Number:   pr.Number,
		URL:      pr.URL,
		Title:    pr.Title,
		IsDraft:  pr.IsDraft,
		Assigned: len(pr.Assignees) > 0,
	}, nil
}

func (g *GitHub) SetDraft(pr *PullRequest, draft bool, dryRun bool) error {
	args := []string{"pr", "ready", strconv.Itoa(pr.Number)}
	if draft {
		args = append(args, "--undo")
	}
	preview := "gh " + strings.Join(args, " ")
	if dryRun {
		ui.Skipped(preview)
		return nil
	}
	ui.Command(preview)
	_, err := g.runRetry("", args...)
	return err
}

func (g *GitHub) AssignSelf(pr *PullRequest, dryRun bool) error {
	args := []string{"pr", "edit", strconv.Itoa(pr.Number), "--add-assignee", selfAssignee}
	preview := "gh " + strings.Join(args, " ")
	if dryRun {
		ui.Skipped(preview)
		return nil
	}
	ui.Command(preview)
	_, err := g.runRetry("", args...)
	return err
}

func (g *GitHub) assignSelfOrWarn(pr *PullRequest) {
	if err := g.AssignSelf(pr, false); err != nil {
		ui.Warn("o PR foi criado, mas não foi possível atribuí-lo a %s: %v", selfAssignee, err)
		ui.Warn("atribua manualmente ou rode a tarefa pr novamente")
	}
}

func (g *GitHub) Create(title, body, base, head string, dryRun bool) (string, error) {
	args := []string{"pr", "create", "--draft", "--base", base, "--head", head, "--title", title, "--body-file", "-"}
	assigned := append(append([]string{}, args...), "--assignee", selfAssignee)
	preview := fmt.Sprintf("gh pr create --draft --base %s --head %s --title %q --body-file - --assignee %s", base, head, title, selfAssignee)
	if dryRun {
		ui.Skipped(preview)
		return "", nil
	}
	ui.Command(preview)
	if strings.TrimSpace(body) == "" {
		body = title
	}
	out, err := g.runRetry(body, assigned...)
	if err == nil {
		return out, nil
	}
	if existing, _ := g.Find(head); existing != nil {
		ui.Warn("o PR #%d foi criado apesar do erro reportado pelo gh", existing.Number)
		if !existing.Assigned {
			g.assignSelfOrWarn(existing)
		}
		return existing.URL, nil
	}
	ui.Warn("falha ao criar o PR atribuído a %s (%v)", selfAssignee, err)
	ui.Warn("criando o PR sem assignee e atribuindo em seguida")
	out, err = g.runRetry(body, args...)
	if err != nil {
		return out, err
	}
	created, _ := g.Find(head)
	if created == nil {
		ui.Warn("o PR foi criado, mas não foi localizado para atribuí-lo a %s; atribua manualmente", selfAssignee)
		return out, nil
	}
	g.assignSelfOrWarn(created)
	return out, nil
}
