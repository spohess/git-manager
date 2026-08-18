package claude

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

const AllowedTools = "Read,Edit,Bash,Git"

const (
	PromptMessage = `@pr-message gere a mensagem das alterações no formato markdown`
	PromptReview  = `@pr-reviewer revise o pr da branch atual`
	PromptFix     = `@pr-comment-fixer corrija os apontamentos feitos no pr a branch corrente mesmo que forem problemas pré-existentes, pode corrigir`
)

func Available() error {
	if _, err := exec.LookPath("claude"); err != nil {
		return fmt.Errorf("comando \"claude\" não encontrado no PATH")
	}
	return nil
}

func command(dir, prompt string) *exec.Cmd {
	cmd := exec.Command("claude", "-p", prompt, "--allowedTools", AllowedTools)
	cmd.Dir = dir
	cmd.Stdin = nil
	return cmd
}

func Capture(dir, prompt string) (string, error) {
	cmd := command(dir, prompt)
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return stdout.String(), fmt.Errorf("claude -p %q falhou: %w", prompt, err)
	}
	return stdout.String(), nil
}

func Stream(dir, prompt string) error {
	cmd := command(dir, prompt)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("claude -p %q falhou: %w", prompt, err)
	}
	return nil
}

func Describe(prompt string) string {
	return fmt.Sprintf("claude -p %q --allowedTools %q", prompt, AllowedTools)
}

type Message struct {
	Title string
	Body  string
}

func ParseMessage(raw string) (Message, error) {
	lines := strings.Split(strings.ReplaceAll(raw, "\r\n", "\n"), "\n")
	if message, ok := parseLabeled(lines); ok {
		return message, nil
	}
	if message, ok := parseBlocks(lines); ok {
		return message, nil
	}
	return Message{}, fmt.Errorf("não foi possível extrair título e descrição da resposta do claude; esperado as seções \"title:\"/\"message:\" ou dois blocos ``` com título e descrição")
}

func parseLabeled(lines []string) (Message, bool) {
	titleAt := findHeader(lines, 0, "title", "título", "titulo")
	if titleAt < 0 {
		return Message{}, false
	}
	bodyAt := findHeader(lines, titleAt+1, "message", "mensagem", "body", "description", "descrição", "descricao")
	titleEnd := len(lines)
	if bodyAt >= 0 {
		titleEnd = bodyAt
	}
	title := firstNonEmpty(extractBlock(lines[titleAt+1 : titleEnd]))
	if title == "" {
		return Message{}, false
	}
	var body string
	if bodyAt >= 0 {
		body = extractBlock(lines[bodyAt+1:])
	}
	return Message{Title: title, Body: body}, true
}

func parseBlocks(lines []string) (Message, bool) {
	fences := fenceIndexes(lines)
	if len(fences) < 2 {
		return Message{}, false
	}
	first := lines[fences[0]+1 : fences[1]]
	at := firstNonEmptyIndex(first)
	if at < 0 {
		return Message{}, false
	}
	title := clean(first[at])
	if title == "" {
		return Message{}, false
	}
	body := strings.Trim(strings.Join(first[at+1:], "\n"), "\n")
	if len(fences) >= 3 {
		start := fences[2] + 1
		end := len(lines)
		if last := fences[len(fences)-1]; last > start {
			end = last
		}
		body = strings.Trim(strings.Join(lines[start:end], "\n"), "\n")
	}
	return Message{Title: title, Body: body}, true
}

func fenceIndexes(lines []string) []int {
	var indexes []int
	for i, line := range lines {
		if isFence(line) {
			indexes = append(indexes, i)
		}
	}
	return indexes
}

func findHeader(lines []string, from int, keywords ...string) int {
	for i := from; i < len(lines); i++ {
		normalized := normalizeHeader(lines[i])
		if normalized == "" {
			continue
		}
		for _, keyword := range keywords {
			if normalized == keyword {
				return i
			}
		}
	}
	return -1
}

func normalizeHeader(line string) string {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" || isFence(trimmed) {
		return ""
	}
	trimmed = strings.Trim(trimmed, "#*_-> \t")
	trimmed = strings.TrimSuffix(strings.TrimSpace(trimmed), ":")
	trimmed = strings.Trim(trimmed, "#*_ \t")
	return strings.ToLower(strings.TrimSpace(trimmed))
}

func extractBlock(lines []string) string {
	start, end := -1, -1
	for i, line := range lines {
		if !isFence(line) {
			continue
		}
		if start < 0 {
			start = i
			continue
		}
		end = i
	}
	if start >= 0 && end > start {
		return strings.Trim(strings.Join(lines[start+1:end], "\n"), "\n")
	}
	return strings.Trim(strings.Join(lines, "\n"), "\n \t")
}

func isFence(line string) bool {
	trimmed := strings.TrimSpace(line)
	return strings.HasPrefix(trimmed, "```") || strings.HasPrefix(trimmed, "~~~")
}

func firstNonEmpty(text string) string {
	lines := strings.Split(text, "\n")
	if at := firstNonEmptyIndex(lines); at >= 0 {
		return clean(lines[at])
	}
	return ""
}

func firstNonEmptyIndex(lines []string) int {
	for i, line := range lines {
		if clean(line) != "" {
			return i
		}
	}
	return -1
}

func clean(line string) string {
	return strings.TrimSpace(strings.Trim(strings.TrimSpace(line), "#*`\"' \t"))
}
