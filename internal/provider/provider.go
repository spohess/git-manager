package provider

import (
	"fmt"
	"strings"
	"time"

	"git-manager/internal/ui"
)

const (
	KindGitHub    = "github"
	KindBitbucket = "bitbucket"

	retryAttempts = 3
	retryDelay    = 2 * time.Second
)

type PullRequest struct {
	Number   int
	URL      string
	Title    string
	IsDraft  bool
	Assigned bool
}

type Provider interface {
	Find(branch string) (*PullRequest, error)
	Create(title, body, base, head string, dryRun bool) (string, error)
	SetDraft(pr *PullRequest, draft bool, dryRun bool) error
	AssignSelf(pr *PullRequest, dryRun bool) error
}

func Available(kind, bitbucketToken string) error {
	switch kind {
	case KindGitHub:
		return githubAvailable()
	case KindBitbucket:
		if strings.TrimSpace(bitbucketToken) == "" {
			return fmt.Errorf("token do Bitbucket não configurado: defina a chave \"bitbucket.token\" no arquivo de configuração ou a variável %s", BitbucketTokenEnv)
		}
		return nil
	}
	return fmt.Errorf("provedor desconhecido: %q", kind)
}

func New(kind, dir, remoteURL, bitbucketToken string) (Provider, error) {
	switch kind {
	case KindGitHub:
		return NewGitHub(dir), nil
	case KindBitbucket:
		return NewBitbucket(remoteURL, bitbucketToken)
	}
	return nil, fmt.Errorf("provedor desconhecido: %q", kind)
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

func withRetry(service string, call func() error) error {
	var err error
	for attempt := 1; attempt <= retryAttempts; attempt++ {
		err = call()
		if err == nil || !Transient(err) {
			return err
		}
		if attempt < retryAttempts {
			ui.Warn("%s indisponível (tentativa %d de %d), repetindo em %s", service, attempt, retryAttempts, retryDelay)
			time.Sleep(retryDelay)
		}
	}
	return err
}
