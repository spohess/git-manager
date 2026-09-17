package provider

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"git-manager/internal/ui"
)

const (
	BitbucketTokenEnv = "BITBUCKET_TOKEN"
	bitbucketHost     = "bitbucket.org"
	bitbucketAPI      = "https://api.bitbucket.org/2.0"
	bitbucketTimeout  = 30 * time.Second
)

type Bitbucket struct {
	workspace string
	slug      string
	token     string
	api       string
	client    *http.Client
}

type bitbucketBranch struct {
	Name string `json:"name"`
}

type bitbucketRef struct {
	Branch bitbucketBranch `json:"branch"`
}

type bitbucketLink struct {
	Href string `json:"href"`
}

type bitbucketPullRequest struct {
	ID    int    `json:"id"`
	Title string `json:"title"`
	State string `json:"state"`
	Draft bool   `json:"draft"`
	Links struct {
		HTML bitbucketLink `json:"html"`
	} `json:"links"`
}

type bitbucketPage struct {
	Values []bitbucketPullRequest `json:"values"`
}

type bitbucketCreate struct {
	Title       string       `json:"title"`
	Description string       `json:"description"`
	Source      bitbucketRef `json:"source"`
	Destination bitbucketRef `json:"destination"`
	Draft       bool         `json:"draft"`
}

type bitbucketUpdate struct {
	Title string `json:"title"`
	Draft bool   `json:"draft"`
}

type bitbucketError struct {
	Error struct {
		Message string `json:"message"`
	} `json:"error"`
}

func NewBitbucket(remoteURL, token string) (*Bitbucket, error) {
	workspace, slug, err := ParseBitbucketRemote(remoteURL)
	if err != nil {
		return nil, err
	}
	return &Bitbucket{
		workspace: workspace,
		slug:      slug,
		token:     strings.TrimSpace(token),
		api:       bitbucketAPI,
		client:    &http.Client{Timeout: bitbucketTimeout},
	}, nil
}

func ParseBitbucketRemote(remoteURL string) (workspace, slug string, err error) {
	remoteURL = strings.TrimSpace(remoteURL)
	index := strings.Index(remoteURL, bitbucketHost)
	if index < 0 {
		return "", "", fmt.Errorf("o remote %q não aponta para %s", remoteURL, bitbucketHost)
	}
	path := strings.TrimLeft(remoteURL[index+len(bitbucketHost):], ":/")
	path = strings.TrimSuffix(strings.TrimSuffix(path, "/"), ".git")
	parts := strings.Split(path, "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("não foi possível extrair workspace/repositório do remote %q", remoteURL)
	}
	return parts[0], parts[1], nil
}

func (b *Bitbucket) Repository() string {
	return b.workspace + "/" + b.slug
}

func (b *Bitbucket) pullRequestsURL() string {
	return fmt.Sprintf("%s/repositories/%s/%s/pullrequests", b.api, b.workspace, b.slug)
}

func (b *Bitbucket) pullRequestURL(number int) string {
	return b.pullRequestsURL() + "/" + strconv.Itoa(number)
}

func (b *Bitbucket) request(method, endpoint string, payload any, out any) error {
	var body io.Reader
	if payload != nil {
		encoded, err := json.Marshal(payload)
		if err != nil {
			return err
		}
		body = bytes.NewReader(encoded)
	}
	req, err := http.NewRequest(method, endpoint, body)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+b.token)
	req.Header.Set("Accept", "application/json")
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := b.client.Do(req)
	if err != nil {
		return fmt.Errorf("%s %s falhou: %w", method, endpoint, err)
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("%s %s falhou: %w", method, endpoint, err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("%s %s falhou: HTTP %d: %s", method, endpoint, resp.StatusCode, bitbucketMessage(data))
	}
	if out == nil || len(data) == 0 {
		return nil
	}
	if err := json.Unmarshal(data, out); err != nil {
		return fmt.Errorf("resposta inesperada do bitbucket em %s %s: %w", method, endpoint, err)
	}
	return nil
}

func (b *Bitbucket) requestRetry(method, endpoint string, payload any, out any) error {
	return withRetry("bitbucket", func() error {
		return b.request(method, endpoint, payload, out)
	})
}

func bitbucketMessage(data []byte) string {
	var parsed bitbucketError
	if err := json.Unmarshal(data, &parsed); err == nil && parsed.Error.Message != "" {
		return parsed.Error.Message
	}
	text := strings.TrimSpace(string(data))
	if text == "" {
		return "sem detalhes"
	}
	return text
}

func convertBitbucket(pr bitbucketPullRequest) *PullRequest {
	return &PullRequest{
		Number:   pr.ID,
		URL:      pr.Links.HTML.Href,
		Title:    pr.Title,
		IsDraft:  pr.Draft,
		Assigned: true,
	}
}

func (b *Bitbucket) Find(branch string) (*PullRequest, error) {
	branch = strings.TrimSpace(branch)
	if branch == "" {
		return nil, fmt.Errorf("branch não informada para localizar o PR no bitbucket")
	}
	query := url.Values{}
	query.Set("state", "OPEN")
	query.Set("q", fmt.Sprintf(`source.branch.name = "%s"`, branch))
	query.Set("pagelen", "1")
	var page bitbucketPage
	if err := b.requestRetry(http.MethodGet, b.pullRequestsURL()+"?"+query.Encode(), nil, &page); err != nil {
		return nil, err
	}
	for _, pr := range page.Values {
		if strings.EqualFold(pr.State, "OPEN") {
			return convertBitbucket(pr), nil
		}
	}
	return nil, nil
}

func (b *Bitbucket) Create(title, body, base, head string, dryRun bool) (string, error) {
	preview := fmt.Sprintf("POST %s (draft, base %s, head %s, título %q)", b.pullRequestsURL(), base, head, title)
	if dryRun {
		ui.Skipped(preview)
		return "", nil
	}
	ui.Command(preview)
	if strings.TrimSpace(body) == "" {
		body = title
	}
	payload := bitbucketCreate{
		Title:       title,
		Description: body,
		Source:      bitbucketRef{Branch: bitbucketBranch{Name: head}},
		Destination: bitbucketRef{Branch: bitbucketBranch{Name: base}},
		Draft:       true,
	}
	var created bitbucketPullRequest
	if err := b.requestRetry(http.MethodPost, b.pullRequestsURL(), payload, &created); err != nil {
		return "", err
	}
	return created.Links.HTML.Href, nil
}

func (b *Bitbucket) SetDraft(pr *PullRequest, draft bool, dryRun bool) error {
	preview := fmt.Sprintf("PUT %s (draft: %t)", b.pullRequestURL(pr.Number), draft)
	if dryRun {
		ui.Skipped(preview)
		return nil
	}
	ui.Command(preview)
	payload := bitbucketUpdate{Title: pr.Title, Draft: draft}
	return b.requestRetry(http.MethodPut, b.pullRequestURL(pr.Number), payload, nil)
}

func (b *Bitbucket) AssignSelf(pr *PullRequest, dryRun bool) error {
	return nil
}
