package git

import (
	"bytes"
	"fmt"
	"os/exec"
	"strconv"
	"strings"

	"git-manager/internal/ui"
)

const DetachedHead = "HEAD"

type Client struct {
	Dir    string
	Remote string
	DryRun bool
}

func New(dir string, dryRun bool) *Client {
	return &Client{Dir: dir, Remote: "origin", DryRun: dryRun}
}

type CommandError struct {
	Args   []string
	Err    error
	Output string
}

func (e *CommandError) Error() string {
	detail := strings.TrimSpace(e.Output)
	if detail == "" {
		return fmt.Sprintf("git %s falhou: %v", strings.Join(e.Args, " "), e.Err)
	}
	return fmt.Sprintf("git %s falhou: %s", strings.Join(e.Args, " "), firstLine(detail))
}

func (e *CommandError) Unwrap() error { return e.Err }

func firstLine(text string) string {
	lines := strings.Split(text, "\n")
	for _, line := range lines {
		if trimmed := strings.TrimSpace(line); trimmed != "" {
			if len(lines) > 1 {
				return trimmed + " (...)"
			}
			return trimmed
		}
	}
	return text
}

func (c *Client) capture(args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = c.Dir
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return strings.TrimSpace(stdout.String()), &CommandError{Args: args, Err: err, Output: stderr.String()}
	}
	return strings.TrimSpace(stdout.String()), nil
}

func (c *Client) succeeds(args ...string) bool {
	_, err := c.capture(args...)
	return err == nil
}

func display(args []string) string {
	parts := make([]string, 0, len(args)+1)
	parts = append(parts, "git")
	for _, arg := range args {
		if arg == "" || strings.ContainsAny(arg, " \t\"'") {
			arg = fmt.Sprintf("%q", arg)
		}
		parts = append(parts, arg)
	}
	return strings.Join(parts, " ")
}

func (c *Client) mutate(args ...string) error {
	line := display(args)
	if c.DryRun {
		ui.Skipped(line)
		return nil
	}
	ui.Command(line)
	cmd := exec.Command("git", args...)
	cmd.Dir = c.Dir
	var combined bytes.Buffer
	cmd.Stdout = &combined
	cmd.Stderr = &combined
	if err := cmd.Run(); err != nil {
		return &CommandError{Args: args, Err: err, Output: combined.String()}
	}
	ui.Output(combined.String())
	return nil
}

func (c *Client) IsRepo() bool {
	out, err := c.capture("rev-parse", "--is-inside-work-tree")
	return err == nil && out == "true"
}

func (c *Client) HasRemote() bool {
	return c.succeeds("remote", "get-url", c.Remote)
}

func (c *Client) CurrentBranch() (string, error) {
	return c.capture("rev-parse", "--abbrev-ref", "HEAD")
}

func (c *Client) IsDirty() (bool, error) {
	out, err := c.capture("status", "--porcelain")
	if err != nil {
		return false, err
	}
	return out != "", nil
}

func (c *Client) DefaultBranch() string {
	if out, err := c.capture("symbolic-ref", "--quiet", "--short", "refs/remotes/"+c.Remote+"/HEAD"); err == nil {
		if name := strings.TrimPrefix(out, c.Remote+"/"); name != "" {
			return name
		}
	}
	for _, name := range []string{"main", "master"} {
		if c.RemoteBranchExists(name) {
			return name
		}
	}
	return "main"
}

func (c *Client) BranchExists(branch string) bool {
	return c.succeeds("rev-parse", "--verify", "--quiet", "refs/heads/"+branch)
}

func (c *Client) RemoteBranchExists(branch string) bool {
	return c.succeeds("show-ref", "--verify", "--quiet", "refs/remotes/"+c.Remote+"/"+branch)
}

func (c *Client) Fetch() error {
	return c.mutate("fetch", "--prune", c.Remote)
}

func (c *Client) StageAll() error {
	return c.mutate("add", "--all")
}

func (c *Client) Commit(message string) error {
	return c.mutate("commit", "--no-verify", "-m", message)
}

func (c *Client) Checkout(branch string) error {
	return c.mutate("checkout", branch)
}

func (c *Client) ForceCheckout(branch string) error {
	return c.mutate("checkout", "--force", branch)
}

func (c *Client) CheckoutNew(branch string) error {
	return c.mutate("checkout", "-b", branch)
}

func (c *Client) CheckoutTracking(branch string) error {
	return c.mutate("checkout", "-b", branch, "--track", c.Remote+"/"+branch)
}

func (c *Client) Pull(branch string) error {
	return c.mutate("pull", c.Remote, branch)
}

func (c *Client) Merge(branch string) error {
	return c.mutate("merge", "--no-edit", branch)
}

func (c *Client) ResetHardRemote(branch string) error {
	return c.mutate("reset", "--hard", c.Remote+"/"+branch)
}

func (c *Client) Push(branch string) error {
	return c.mutate("push", "--set-upstream", c.Remote, branch)
}

func (c *Client) ForcePush(branch string) error {
	return c.mutate("push", "--force", "--set-upstream", c.Remote, branch)
}

func (c *Client) GoneBranches() ([]string, error) {
	out, err := c.capture("for-each-ref", "--format=%(refname:short)%09%(upstream:track)", "refs/heads")
	if err != nil {
		return nil, err
	}
	var branches []string
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "\t", 2)
		if len(parts) != 2 {
			continue
		}
		if strings.Contains(parts[1], "[gone]") {
			branches = append(branches, parts[0])
		}
	}
	return branches, nil
}

func (c *Client) DeleteBranch(branch string) error {
	return c.mutate("branch", "-D", branch)
}

func (c *Client) HasCommitsAhead(base, branch string) bool {
	out, err := c.capture("rev-list", "--count", c.Remote+"/"+base+".."+branch)
	if err != nil {
		return true
	}
	return out != "0"
}

type StatusInfo struct {
	Branch    string
	Detached  bool
	Upstream  string
	Ahead     int
	Behind    int
	Staged    int
	Unstaged  int
	Untracked int
	Conflicts int
}

func (s StatusInfo) Changed() int {
	return s.Staged + s.Unstaged + s.Untracked + s.Conflicts
}

func (s StatusInfo) Clean() bool {
	return s.Changed() == 0
}

func (c *Client) Status() (StatusInfo, error) {
	out, err := c.capture("status", "--porcelain=v2", "--branch")
	if err != nil {
		return StatusInfo{}, err
	}

	var info StatusInfo
	for _, line := range strings.Split(out, "\n") {
		if line == "" {
			continue
		}
		switch {
		case strings.HasPrefix(line, "# branch.head "):
			info.Branch = strings.TrimPrefix(line, "# branch.head ")
			info.Detached = info.Branch == "(detached)"
		case strings.HasPrefix(line, "# branch.upstream "):
			info.Upstream = strings.TrimPrefix(line, "# branch.upstream ")
		case strings.HasPrefix(line, "# branch.ab "):
			for _, field := range strings.Fields(strings.TrimPrefix(line, "# branch.ab ")) {
				n, err := strconv.Atoi(strings.TrimLeft(field, "+-"))
				if err != nil {
					continue
				}
				if strings.HasPrefix(field, "+") {
					info.Ahead = n
				} else if strings.HasPrefix(field, "-") {
					info.Behind = n
				}
			}
		case strings.HasPrefix(line, "1 "), strings.HasPrefix(line, "2 "):
			fields := strings.Fields(line)
			if len(fields) < 2 || len(fields[1]) != 2 {
				continue
			}
			if fields[1][0] != '.' {
				info.Staged++
			}
			if fields[1][1] != '.' {
				info.Unstaged++
			}
		case strings.HasPrefix(line, "u "):
			info.Conflicts++
		case strings.HasPrefix(line, "? "):
			info.Untracked++
		}
	}
	return info, nil
}

func (c *Client) AheadBehind(base, branch string) (ahead, behind int, err error) {
	out, err := c.capture("rev-list", "--left-right", "--count", base+"..."+branch)
	if err != nil {
		return 0, 0, err
	}
	fields := strings.Fields(out)
	if len(fields) != 2 {
		return 0, 0, fmt.Errorf("saída inesperada de rev-list --left-right --count: %q", out)
	}
	if behind, err = strconv.Atoi(fields[0]); err != nil {
		return 0, 0, err
	}
	if ahead, err = strconv.Atoi(fields[1]); err != nil {
		return 0, 0, err
	}
	return ahead, behind, nil
}

func Available() error {
	if _, err := exec.LookPath("git"); err != nil {
		return fmt.Errorf("comando \"git\" não encontrado no PATH")
	}
	return nil
}
