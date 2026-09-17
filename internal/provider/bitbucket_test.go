package provider

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestParseBitbucketRemote(t *testing.T) {
	cases := map[string][2]string{
		"git@bitbucket.org:acme/backend.git":          {"acme", "backend"},
		"ssh://git@bitbucket.org/acme/backend.git":    {"acme", "backend"},
		"https://user@bitbucket.org/acme/backend.git": {"acme", "backend"},
		"https://bitbucket.org/acme/backend":          {"acme", "backend"},
		"https://bitbucket.org/acme/backend/":         {"acme", "backend"},
	}
	for remote, want := range cases {
		workspace, slug, err := ParseBitbucketRemote(remote)
		if err != nil {
			t.Errorf("%s: %v", remote, err)
			continue
		}
		if workspace != want[0] || slug != want[1] {
			t.Errorf("%s: esperado %s/%s, obtido %s/%s", remote, want[0], want[1], workspace, slug)
		}
	}
}

func TestParseBitbucketRemoteInvalid(t *testing.T) {
	for _, remote := range []string{"git@github.com:acme/backend.git", "https://bitbucket.org/acme", ""} {
		if _, _, err := ParseBitbucketRemote(remote); err == nil {
			t.Errorf("%q: esperado erro", remote)
		}
	}
}

type recorded struct {
	method string
	path   string
	query  string
	user   string
	pass   string
	body   map[string]any
}

func newTestBitbucket(t *testing.T, handler func(w http.ResponseWriter, r recorded)) (*Bitbucket, *[]recorded) {
	t.Helper()
	var calls []recorded
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		call := recorded{method: r.Method, path: r.URL.Path, query: r.URL.RawQuery}
		call.user, call.pass, _ = r.BasicAuth()
		if r.Body != nil {
			_ = json.NewDecoder(r.Body).Decode(&call.body)
		}
		calls = append(calls, call)
		handler(w, call)
	}))
	t.Cleanup(server.Close)
	bb, err := NewBitbucket("git@bitbucket.org:acme/backend.git", "dev@acme.com", "secreto")
	if err != nil {
		t.Fatalf("NewBitbucket: %v", err)
	}
	bb.api = server.URL
	return bb, &calls
}

func TestBitbucketFind(t *testing.T) {
	bb, calls := newTestBitbucket(t, func(w http.ResponseWriter, r recorded) {
		_, _ = w.Write([]byte(`{"values":[{"id":42,"title":"Login","state":"OPEN","draft":true,"links":{"html":{"href":"https://bitbucket.org/acme/backend/pull-requests/42"}}}]}`))
	})
	pr, err := bb.Find("feature/login")
	if err != nil {
		t.Fatalf("Find: %v", err)
	}
	if pr == nil || pr.Number != 42 || !pr.IsDraft || pr.Title != "Login" || !pr.Assigned {
		t.Fatalf("PR inesperado: %+v", pr)
	}
	if pr.URL != "https://bitbucket.org/acme/backend/pull-requests/42" {
		t.Errorf("URL inesperada: %s", pr.URL)
	}
	call := (*calls)[0]
	if call.method != http.MethodGet || call.path != "/repositories/acme/backend/pullrequests" {
		t.Errorf("chamada inesperada: %s %s", call.method, call.path)
	}
	if call.user != "dev@acme.com" || call.pass != "secreto" {
		t.Errorf("basic auth inesperado: %s:%s", call.user, call.pass)
	}
	if !strings.Contains(call.query, "state=OPEN") || !strings.Contains(call.query, "source.branch.name") {
		t.Errorf("query inesperada: %s", call.query)
	}
}

func TestBitbucketFindNotFound(t *testing.T) {
	bb, _ := newTestBitbucket(t, func(w http.ResponseWriter, r recorded) {
		_, _ = w.Write([]byte(`{"values":[]}`))
	})
	pr, err := bb.Find("feature/login")
	if err != nil {
		t.Fatalf("Find: %v", err)
	}
	if pr != nil {
		t.Fatalf("esperado nil, obtido %+v", pr)
	}
}

func TestBitbucketFindError(t *testing.T) {
	bb, _ := newTestBitbucket(t, func(w http.ResponseWriter, r recorded) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"type":"error","error":{"message":"Token is invalid"}}`))
	})
	_, err := bb.Find("feature/login")
	if err == nil || !strings.Contains(err.Error(), "Token is invalid") {
		t.Fatalf("esperado erro com a mensagem do bitbucket, obtido %v", err)
	}
}

func TestBitbucketCreate(t *testing.T) {
	bb, calls := newTestBitbucket(t, func(w http.ResponseWriter, r recorded) {
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"id":7,"links":{"html":{"href":"https://bitbucket.org/acme/backend/pull-requests/7"}}}`))
	})
	url, err := bb.Create("Título", "Descrição", "main", "feature/login", false)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if url != "https://bitbucket.org/acme/backend/pull-requests/7" {
		t.Errorf("URL inesperada: %s", url)
	}
	call := (*calls)[0]
	if call.method != http.MethodPost || call.path != "/repositories/acme/backend/pullrequests" {
		t.Errorf("chamada inesperada: %s %s", call.method, call.path)
	}
	if call.body["title"] != "Título" || call.body["description"] != "Descrição" || call.body["draft"] != true {
		t.Errorf("corpo inesperado: %+v", call.body)
	}
	source := call.body["source"].(map[string]any)["branch"].(map[string]any)["name"]
	destination := call.body["destination"].(map[string]any)["branch"].(map[string]any)["name"]
	if source != "feature/login" || destination != "main" {
		t.Errorf("branches inesperadas: %v -> %v", source, destination)
	}
}

func TestBitbucketCreateDryRun(t *testing.T) {
	bb, calls := newTestBitbucket(t, func(w http.ResponseWriter, r recorded) {
		t.Fatal("nenhuma requisição deveria ser feita em dry-run")
	})
	if _, err := bb.Create("Título", "", "main", "feature/login", true); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if len(*calls) != 0 {
		t.Fatalf("esperado 0 chamadas, obtido %d", len(*calls))
	}
}

func TestBitbucketSetDraft(t *testing.T) {
	bb, calls := newTestBitbucket(t, func(w http.ResponseWriter, r recorded) {
		_, _ = w.Write([]byte(`{"id":42,"draft":false}`))
	})
	pr := &PullRequest{Number: 42, Title: "Login", IsDraft: true}
	if err := bb.SetDraft(pr, false, false); err != nil {
		t.Fatalf("SetDraft: %v", err)
	}
	call := (*calls)[0]
	if call.method != http.MethodPut || call.path != "/repositories/acme/backend/pullrequests/42" {
		t.Errorf("chamada inesperada: %s %s", call.method, call.path)
	}
	if call.body["draft"] != false || call.body["title"] != "Login" {
		t.Errorf("corpo inesperado: %+v", call.body)
	}
}
