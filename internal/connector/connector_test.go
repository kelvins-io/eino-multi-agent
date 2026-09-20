package connector

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/kelvins-io/eino-multi-agent/internal/store"
)

func TestInvokeWebhook(t *testing.T) {
	var got map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Harness-Secret") != "s3cret" {
			t.Errorf("missing secret")
		}
		b, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(b, &got)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()
	item := &store.Connector{
		Kind:   KindWebhook,
		Config: map[string]string{"url": srv.URL, "secret": "s3cret"},
	}
	err := invokeWebhook(context.Background(), srv.Client(), item, Event{
		Task: &store.Task{ID: "t1", Title: "周报", Status: store.StatusSucceeded},
	})
	if err != nil {
		t.Fatal(err)
	}
	if got["event"] != "task.succeeded" {
		t.Fatalf("%#v", got)
	}
}
