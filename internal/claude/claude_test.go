package claude

import "testing"

func TestParseMessage(t *testing.T) {
	raw := "title:\n```\nTítulo de exemplo do PR\n```\n\nmessage:\n```\nMensagem de exemplo do PR\n```\n"
	msg, err := ParseMessage(raw)
	if err != nil {
		t.Fatalf("ParseMessage: %v", err)
	}
	if msg.Title != "Título de exemplo do PR" {
		t.Errorf("Title = %q", msg.Title)
	}
	if msg.Body != "Mensagem de exemplo do PR" {
		t.Errorf("Body = %q", msg.Body)
	}
}

func TestParseMessageComRuidoEBlocosAninhados(t *testing.T) {
	raw := "Analisando as alterações...\n\n**title:**\n```markdown\nfeat: adiciona login\n```\n\n**message:**\n```markdown\n## Resumo\n\nAdiciona o fluxo de login.\n\n```go\nfunc Login() {}\n```\n\n- item final\n```\n"
	msg, err := ParseMessage(raw)
	if err != nil {
		t.Fatalf("ParseMessage: %v", err)
	}
	if msg.Title != "feat: adiciona login" {
		t.Errorf("Title = %q", msg.Title)
	}
	if want := "## Resumo\n\nAdiciona o fluxo de login.\n\n```go\nfunc Login() {}\n```\n\n- item final"; msg.Body != want {
		t.Errorf("Body = %q, esperado %q", msg.Body, want)
	}
}

func TestParseMessageSemCercas(t *testing.T) {
	raw := "title:\nfeat: ajusta build\n\nmessage:\nDescrição simples\n"
	msg, err := ParseMessage(raw)
	if err != nil {
		t.Fatalf("ParseMessage: %v", err)
	}
	if msg.Title != "feat: ajusta build" {
		t.Errorf("Title = %q", msg.Title)
	}
	if msg.Body != "Descrição simples" {
		t.Errorf("Body = %q", msg.Body)
	}
}

func TestParseMessageSemRotulos(t *testing.T) {
	raw := "Aqui está a mensagem do PR pronta para copiar e colar:\n\n```\nRefatorar resolução de veterinário no webhook\n```\n\n```\n## Resumo\nRefatora a resolução de veterinários.\n\n## Notas\n- sem breaking changes\n```\n"
	msg, err := ParseMessage(raw)
	if err != nil {
		t.Fatalf("ParseMessage: %v", err)
	}
	if msg.Title != "Refatorar resolução de veterinário no webhook" {
		t.Errorf("Title = %q", msg.Title)
	}
	if want := "## Resumo\nRefatora a resolução de veterinários.\n\n## Notas\n- sem breaking changes"; msg.Body != want {
		t.Errorf("Body = %q, esperado %q", msg.Body, want)
	}
}

func TestParseMessageBlocoUnico(t *testing.T) {
	raw := "```\nCorrigir o webhook\n\n## Resumo\nAjusta o retorno 202.\n```\n"
	msg, err := ParseMessage(raw)
	if err != nil {
		t.Fatalf("ParseMessage: %v", err)
	}
	if msg.Title != "Corrigir o webhook" {
		t.Errorf("Title = %q", msg.Title)
	}
	if want := "## Resumo\nAjusta o retorno 202."; msg.Body != want {
		t.Errorf("Body = %q", msg.Body)
	}
}

func TestParseMessageInvalida(t *testing.T) {
	for name, raw := range map[string]string{
		"sem seções e sem blocos": "resposta qualquer em texto corrido",
		"vazio":                   "title:\n```\n```\n",
	} {
		if _, err := ParseMessage(raw); err == nil {
			t.Errorf("%s: esperado erro", name)
		}
	}
}
