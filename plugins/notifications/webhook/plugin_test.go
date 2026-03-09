package webhook

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/luminarr/luminarr/pkg/plugin"
)

func TestNotify_Success(t *testing.T) {
	var gotEvent plugin.NotificationEvent
	var gotHeaders http.Header
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		gotHeaders = r.Header
		body, _ := io.ReadAll(r.Body)
		if err := json.Unmarshal(body, &gotEvent); err != nil {
			t.Fatalf("unmarshal body: %v", err)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	n := &Notifier{
		cfg: Config{
			URL:     srv.URL,
			Method:  http.MethodPost,
			Headers: map[string]string{"X-Custom": "test-value"},
		},
		client: srv.Client(),
	}

	event := plugin.NotificationEvent{
		Type:      plugin.EventDownloadDone,
		Timestamp: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		Message:   "Download complete: Inception",
	}

	if err := n.Notify(context.Background(), event); err != nil {
		t.Fatalf("Notify() = %v", err)
	}

	if gotEvent.Message != "Download complete: Inception" {
		t.Errorf("message = %q", gotEvent.Message)
	}
	if gotHeaders.Get("X-Custom") != "test-value" {
		t.Errorf("X-Custom header = %q, want test-value", gotHeaders.Get("X-Custom"))
	}
	if gotHeaders.Get("Content-Type") != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", gotHeaders.Get("Content-Type"))
	}
}

func TestNotify_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	n := &Notifier{
		cfg:    Config{URL: srv.URL, Method: http.MethodPost},
		client: srv.Client(),
	}

	err := n.Notify(context.Background(), plugin.NotificationEvent{Type: "test", Message: "hi"})
	if err == nil {
		t.Fatal("expected error for 500 response")
	}
}

func TestNew_DefaultMethod(t *testing.T) {
	n := New(Config{URL: "http://example.com"})
	if n.cfg.Method != http.MethodPost {
		t.Errorf("Method = %q, want POST", n.cfg.Method)
	}
}

func TestNotify_CustomMethod(t *testing.T) {
	var gotMethod string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	n := &Notifier{
		cfg:    Config{URL: srv.URL, Method: http.MethodPut},
		client: srv.Client(),
	}

	if err := n.Notify(context.Background(), plugin.NotificationEvent{Type: "test", Message: "hi"}); err != nil {
		t.Fatalf("Notify() = %v", err)
	}
	if gotMethod != http.MethodPut {
		t.Errorf("method = %s, want PUT", gotMethod)
	}
}
