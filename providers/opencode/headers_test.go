package opencode

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestGenerateHeadersNoGit(t *testing.T) {
	dir := t.TempDir()
	headers, err := GenerateHeaders(HeaderOptions{
		Dir:     dir,
		Client:  "test-client",
		Version: "test-version",
		Time:    time.UnixMilli(1700000000000),
	})
	if err != nil {
		t.Fatalf("GenerateHeaders returned error: %v", err)
	}
	if len(headers) != 5 {
		t.Fatalf("expected 5 headers, got %d", len(headers))
	}

	values := map[string]string{}
	for _, header := range headers {
		values[header.Name] = header.Value
	}
	if values[HeaderProject] != DefaultProjectID {
		t.Fatalf("project header mismatch: %q", values[HeaderProject])
	}
	if !strings.HasPrefix(values[HeaderSession], "ses_") {
		t.Fatalf("session header should start with ses_: %q", values[HeaderSession])
	}
	if !strings.HasPrefix(values[HeaderRequest], "msg_") {
		t.Fatalf("request header should start with msg_: %q", values[HeaderRequest])
	}
	if values[HeaderClient] != "test-client" {
		t.Fatalf("client header mismatch: %q", values[HeaderClient])
	}
	if values[HeaderUA] != "opencode/test-version" {
		t.Fatalf("user-agent header mismatch: %q", values[HeaderUA])
	}
}

func TestGenerateHeadersUsesEnvDefaults(t *testing.T) {
	t.Setenv("OPENCODE_CLIENT", "env-client")
	t.Setenv("OPENCODE_VERSION", "1.2.3")

	headers, err := GenerateHeaders(HeaderOptions{Dir: t.TempDir()})
	if err != nil {
		t.Fatalf("GenerateHeaders returned error: %v", err)
	}
	values := map[string]string{}
	for _, header := range headers {
		values[header.Name] = header.Value
	}
	if values[HeaderClient] != "env-client" {
		t.Fatalf("client header mismatch: %q", values[HeaderClient])
	}
	if values[HeaderUA] != "opencode/1.2.3" {
		t.Fatalf("user-agent header mismatch: %q", values[HeaderUA])
	}
}

func TestHeaderJSONShape(t *testing.T) {
	data, err := json.Marshal(struct {
		Headers []Header `json:"headers"`
	}{
		Headers: []Header{{Name: "x-test", Value: "ok"}},
	})
	if err != nil {
		t.Fatalf("Marshal returned error: %v", err)
	}
	if string(data) != `{"headers":[{"name":"x-test","value":"ok"}]}` {
		t.Fatalf("unexpected JSON: %s", data)
	}
}
