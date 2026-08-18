package gh

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"git-manager/internal/ui"
)

const (
	selfAssignee  = "@me"
	retryAttempts = 3
	retryDelay    = 2 * time.Second
)

type User struct {
	Login string `json:"login"`
}

type PullRequest struct {
	Number    int    `json:"number"`
	URL       string `json:"url"`
	State     string `json:"state"`
	Title     string `json:"title"`
	IsDraft   bool   `json:"isDraft"`
	Assignees []User `json:"assignees"`
}

func (pr *PullRequest) HasAssignee() bool {
	return pr != nil && len(pr.Assignees) > 0
}

func Available() error {
	if _, err := exec.LookPath("gh"); err != nil {
		return fmt.Errorf("comando \"gh\" (GitHub CLI) não encontrado no PATH")
	}
	return nil
}

func run(dir string, stdin string, args ...string) (string, error) {
	cmd := exec.Command("gh", args...)
	cmd.Dir = dir
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

func runRetry(dir string, stdin string, args ...string) (string, error) {
	var out string
	var err error
	for attempt := 1; attempt <= retryAttempts; attempt++ {
		out, err = run(dir, stdin, args...)
		if err == nil || !Transient(err) {
			return out, err
		}
		if attempt < retryAttempts {
			ui.Warn("github indisponível (tentativa %d de %d), repetindo em %s", attempt, retryAttempts, retryDelay)
			time.Sleep(retryDelay)
		}
	}
	return out, err
}

var transientMarkers = []string{
	"http 5",
	"no server is currently available",
	"service unavailable",
	"bad gateway",
	"temporarily unavailable",
	"timeout",
	"timed out",
	"connection reset",
	"unexpected eof",
	"try again",
}

func Transient(err error) bool {
	if err == nil {
		return false
	}
	message := strings.ToLower(err.Error())
	for _, marker := range transientMarkers {
		if strings.Contains(message, marker) {
			return true
		}
	}
	return false
}

const viewFields = "number,url,state,title,isDraft,assignees"

func Current(dir string) (*PullRequest, error) {
	return View(dir, "")
}

func View(dir, branch string) (*PullRequest, error) {
	args := []string{"pr", "view"}
	if branch = strings.TrimSpace(branch); branch != "" {
		args = append(args, branch)
	}
	args = append(args, "--json", viewFields)
	out, err := run(dir, "", args...)
	if err != nil {
		return nil, nil
	}
	var pr PullRequest
	if err := json.Unmarshal([]byte(out), &pr); err != nil {
		return nil, fmt.Errorf("resposta inesperada do gh pr view: %w", err)
	}
	if pr.Number == 0 || strings.EqualFold(pr.State, "CLOSED") || strings.EqualFold(pr.State, "MERGED") {
		return nil, nil
	}
	return &pr, nil
}

func MarkDraft(dir, branch string, dryRun bool) error {
	args := []string{"pr", "ready"}
	if branch = strings.TrimSpace(branch); branch != "" {
		args = append(args, branch)
	}
	args = append(args, "--undo")
	preview := "gh " + strings.Join(args, " ")
	if dryRun {
		ui.Skipped(preview)
		return nil
	}
	ui.Command(preview)
	_, err := runRetry(dir, "", args...)
	return err
}

func MarkReady(dir, branch string, dryRun bool) error {
	args := []string{"pr", "ready"}
	if branch = strings.TrimSpace(branch); branch != "" {
		args = append(args, branch)
	}
	preview := "gh " + strings.Join(args, " ")
	if dryRun {
		ui.Skipped(preview)
		return nil
	}
	ui.Command(preview)
	_, err := runRetry(dir, "", args...)
	return err
}

func AssignSelf(dir string, dryRun bool) error {
	preview := fmt.Sprintf("gh pr edit --add-assignee %s", selfAssignee)
	if dryRun {
		ui.Skipped(preview)
		return nil
	}
	ui.Command(preview)
	_, err := runRetry(dir, "", "pr", "edit", "--add-assignee", selfAssignee)
	return err
}

func assignSelfOrWarn(dir string) {
	if err := AssignSelf(dir, false); err != nil {
		ui.Warn("o PR foi criado, mas não foi possível atribuí-lo a %s: %v", selfAssignee, err)
		ui.Warn("atribua manualmente ou rode a tarefa pr novamente")
	}
}

func Create(dir, title, body, base, head string, dryRun bool) (string, error) {
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
	out, err := runRetry(dir, body, assigned...)
	if err == nil {
		return out, nil
	}
	if existing, _ := Current(dir); existing != nil {
		ui.Warn("o PR #%d foi criado apesar do erro reportado pelo gh", existing.Number)
		if !existing.HasAssignee() {
			assignSelfOrWarn(dir)
		}
		return existing.URL, nil
	}
	ui.Warn("falha ao criar o PR atribuído a %s (%v)", selfAssignee, err)
	ui.Warn("criando o PR sem assignee e atribuindo em seguida")
	out, err = runRetry(dir, body, args...)
	if err != nil {
		return out, err
	}
	assignSelfOrWarn(dir)
	return out, nil
}
