package email

import (
	"strings"
	"testing"
)

func TestBuildMessage(t *testing.T) {
	msg := buildMessage("noreply@luminarr.dev", []string{"user@example.com"}, "Test Subject", "Hello body")
	s := string(msg)

	if !strings.Contains(s, "From: noreply@luminarr.dev\r\n") {
		t.Error("missing or wrong From header")
	}
	if !strings.Contains(s, "To: user@example.com\r\n") {
		t.Error("missing or wrong To header")
	}
	if !strings.Contains(s, "Subject: Test Subject\r\n") {
		t.Error("missing or wrong Subject header")
	}
	if !strings.Contains(s, "Content-Type: text/plain; charset=UTF-8\r\n") {
		t.Error("missing Content-Type header")
	}
	if !strings.Contains(s, "Hello body") {
		t.Error("missing body")
	}
}

func TestBuildMessage_MultipleRecipients(t *testing.T) {
	msg := buildMessage("a@b.com", []string{"x@y.com", "z@w.com"}, "Sub", "Body")
	s := string(msg)

	if !strings.Contains(s, "To: x@y.com, z@w.com\r\n") {
		t.Errorf("To header wrong: %s", s)
	}
}

func TestSanitizeHeader_StripsCRLF(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"normal", "normal"},
		{"with\nnewline", "withnewline"},
		{"with\rcarriage", "withcarriage"},
		{"with\r\nboth", "withboth"},
	}
	for _, tt := range tests {
		got := sanitizeHeader(tt.input)
		if got != tt.want {
			t.Errorf("sanitizeHeader(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestNew_DefaultPort(t *testing.T) {
	n := New(Config{Host: "smtp.example.com", From: "a@b.com", To: []string{"x@y.com"}})
	if n.cfg.Port != 587 {
		t.Errorf("Port = %d, want 587", n.cfg.Port)
	}
}

func TestNew_CustomPort(t *testing.T) {
	n := New(Config{Host: "smtp.example.com", Port: 465, From: "a@b.com", To: []string{"x@y.com"}})
	if n.cfg.Port != 465 {
		t.Errorf("Port = %d, want 465", n.cfg.Port)
	}
}
