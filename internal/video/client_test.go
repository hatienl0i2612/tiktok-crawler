package video

import "testing"

func TestNewClientUsesCustomHeaderUserAgent(t *testing.T) {
	t.Parallel()
	client, err := NewClient(ClientOptions{Headers: map[string]string{"User-Agent": "custom-agent"}})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	if client.session.UserAgent() != "custom-agent" {
		t.Fatalf("UserAgent() = %q, want the custom header value", client.session.UserAgent())
	}
}

func TestNewClientUsesVideoDefaults(t *testing.T) {
	t.Parallel()
	client, err := NewClient(ClientOptions{Cookie: "Cookie: session=abc"})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	if client.session.UserAgent() != DefaultUserAgent {
		t.Fatalf("UserAgent() = %q", client.session.UserAgent())
	}
	if client.session.CookieValue("session") != "abc" {
		t.Fatalf("CookieValue() = %q", client.session.CookieValue("session"))
	}
}
