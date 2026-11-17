package a2a

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/kelvins-io/eino-multi-agent/internal/config"
)

func TestAgentCardURL(t *testing.T) {
	got := AgentCardURL(config.A2AConfig{PublicURL: "http://127.0.0.1:8181/"})
	want := "http://127.0.0.1:8181/.well-known/agent-card.json"
	if got != want {
		t.Fatalf("got %s want %s", got, want)
	}
}

func TestRegisterToRegistries(t *testing.T) {
	var gotURI string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method %s", r.Method)
		}
		if r.Header.Get("Authorization") != "Bearer tok" {
			t.Fatalf("auth %q", r.Header.Get("Authorization"))
		}
		var body map[string]string
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		gotURI = body["wellKnownURI"]
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	RegisterToRegistries(context.Background(), config.A2AConfig{
		PublicURL: "http://agent.example:8181",
		Registries: []config.A2ARegistryConfig{
			{URL: srv.URL, AuthToken: "tok"},
		},
	})
	want := "http://agent.example:8181/.well-known/agent-card.json"
	if gotURI != want {
		t.Fatalf("got %q want %q", gotURI, want)
	}
}
