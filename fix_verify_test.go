package main

import (
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"testing"
)

var errTestSentinel = errors.New("test transport error")

func TestFixProxyfileAbsoluteRejected(t *testing.T) {
	cfg := Config{Listen: "x", ServerKeys: []string{"k"}, Anonymous: true, ProxyFile: "/etc/passwd"}
	if _, err := NormalizeConfig("/tmp/c.json", cfg); err == nil {
		t.Fatal("absolute proxyfile accepted")
	}
}

func TestFixProxyfileTraversalRejected(t *testing.T) {
	cfg := Config{Listen: "x", ServerKeys: []string{"k"}, Anonymous: true, ProxyFile: "../secret.txt"}
	if _, err := NormalizeConfig("/tmp/c.json", cfg); err == nil {
		t.Fatal("traversal proxyfile accepted")
	}
}

func TestFixProxyfileRelativeAccepted(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "proxies.txt"), []byte("direct\n"), 0o644)
	cfg := Config{Listen: "0.0.0.0:8080", ServerKeys: []string{"k"}, Anonymous: true, Prefer: TierGo,
		Upstream: UpstreamConfig{Zen: "https://opencode.ai/zen", Go: "https://opencode.ai/zen/go"},
		Retry:    RetryConfig{MaxAttempts: 3, TimeoutSeconds: 300},
		Models:   ModelsConfig{RefreshSeconds: 300},
		Performance: PerformanceConfig{MaxIdleConns: 8, MaxIdleConnsPerHost: 4,
			IdleConnTimeoutSeconds: 60, ConnectTimeoutSeconds: 5, FailureCooldownSeconds: 15},
		Logging: LoggingConfig{Level: "info", RingSize: 200},
		ProxyFile: "proxies.txt", WebUI: WebUIConfig{Enabled: true, Username: "a", Password: "0123456789abcdef", SessionTTLMinutes: 720}}
	if _, err := NormalizeConfig(filepath.Join(dir, "config.json"), cfg); err != nil {
		t.Fatal("legit relative proxyfile rejected:", err)
	}
}

func TestFixWebUIListenRejected(t *testing.T) {
	cfg := Config{Listen: "0.0.0.0:8080", ServerKeys: []string{"k"}, Anonymous: true,
		WebUI: WebUIConfig{Enabled: true, Listen: "127.0.0.1:9999", Username: "a"}}
	if _, err := NormalizeConfig("/tmp/c.json", cfg); err == nil {
		t.Fatal("webui.listen silently accepted")
	}
}

func TestFixObserverBufferCapped(t *testing.T) {
	obs := newStreamUsageObserver(ProtocolChat)
	big := make([]byte, 20<<20)
	for i := range big {
		big[i] = 'x'
	}
	if _, err := obs.Write(big); err != nil {
		t.Fatal("write returned err:", err)
	}
	if len(obs.buffer) > maxSSEObserverBuffer {
		t.Fatalf("buffer unbounded: %d", len(obs.buffer))
	}
	if obs.ParseError() == nil {
		t.Fatal("expected parseErr after overflow")
	}
}

func TestFixResponsesUnknownRoleRejected(t *testing.T) {
	_, err := decodeBridgeRequest(ProtocolResponses, map[string]any{
		"input": []any{map[string]any{"type": "message", "role": "weird", "content": "hi"}},
	})
	if err == nil {
		t.Fatal("unknown role silently dropped")
	}
}

func validTestConfig() Config {
	return Config{Listen: "0.0.0.0:8080", ServerKeys: []string{"k"}, Anonymous: true, Prefer: TierGo,
		Upstream: UpstreamConfig{Zen: "https://opencode.ai/zen", Go: "https://opencode.ai/zen/go"},
		Retry:    RetryConfig{MaxAttempts: 3, TimeoutSeconds: 300},
		Models:   ModelsConfig{RefreshSeconds: 300},
		Performance: PerformanceConfig{MaxIdleConns: 8, MaxIdleConnsPerHost: 4,
			IdleConnTimeoutSeconds: 60, ConnectTimeoutSeconds: 5, FailureCooldownSeconds: 15},
		Logging: LoggingConfig{Level: "info", RingSize: 200},
		WebUI:   WebUIConfig{Enabled: true, Username: "a", Password: "0123456789abcdef", SessionTTLMinutes: 720}}
}

func TestFixListenFormatRejected(t *testing.T) {
	for _, bad := range []string{"8080", "localhost", "0.0.0.0:", ":abc", "http://x:8080/"} {
		cfg := validTestConfig()
		cfg.Listen = bad
		if _, err := NormalizeConfig("/tmp/c.json", cfg); err == nil {
			t.Fatalf("invalid listen %q accepted", bad)
		}
	}
	cfg := validTestConfig()
	if _, err := NormalizeConfig("/tmp/c.json", cfg); err != nil {
		t.Fatal("valid listen rejected:", err)
	}
}

func TestFixDurationBoundsRejected(t *testing.T) {
	cases := []func(*Config){
		func(c *Config) { c.Retry.MaxAttempts = 1000000 },
		func(c *Config) { c.Retry.TimeoutSeconds = 1 << 30 },
		func(c *Config) { c.Models.RefreshSeconds = 1 << 30 },
		func(c *Config) { c.Performance.MaxIdleConns = 1 << 30 },
		func(c *Config) { c.Performance.ConnectTimeoutSeconds = 1 << 30 },
	}
	for i, mutate := range cases {
		cfg := validTestConfig()
		mutate(&cfg)
		if _, err := NormalizeConfig("/tmp/c.json", cfg); err == nil {
			t.Fatalf("case %d: oversized duration/count accepted", i)
		}
	}
}

func TestFixPasswordCheckRateLimited(t *testing.T) {
	a := NewAdminServer(nil, nil, nil, nil)
	client := "10.0.0.9"
	for i := 0; i < 10; i++ {
		if !a.allowPasswordCheck(client) {
			t.Fatalf("blocked too early at %d", i)
		}
		a.recordPasswordFailure(client)
	}
	if a.allowPasswordCheck(client) {
		t.Fatal("not rate limited after 10 failures")
	}
}

func TestFixNonObjectResponseRejected(t *testing.T) {
	_, err := decodeBridgeResponse(ProtocolChat, map[string]any{
		"choices": []any{"oops"},
	})
	if err == nil {
		t.Fatal("non-object choice silently accepted")
	}
	_, err = decodeBridgeResponse(ProtocolChat, map[string]any{
		"choices": []any{map[string]any{
			"message": map[string]any{"content": "hi", "tool_calls": []any{"oops"}},
		}},
	})
	if err == nil {
		t.Fatal("non-object tool_call silently accepted")
	}
}

func TestFixShortKeyRedacted(t *testing.T) {
	if got := keyDisplayID("abc"); got != "***" {
		t.Fatalf("short key leaked: %q", got)
	}
	r := NewSecretRedactor()
	r.Replace(Config{ServerKeys: []string{"k"}})
	if got := r.String("auth k done"); got != "auth *** done" {
		t.Fatalf("short secret not redacted: %q", got)
	}
}

func TestFixAnthropicFileValidation(t *testing.T) {
	req := bridgeRequest{Messages: []bridgeMessage{{Role: "user", Blocks: []bridgeBlock{{Kind: "file", Filename: "a.pdf"}}}}}
	if err := validateBridgeRequest(ProtocolAnthropic, req); err == nil {
		t.Fatal("filename-only file block silently accepted for Anthropic")
	}
}

func TestFixRetryableSemantics(t *testing.T) {
	resp := func(code int) *http.Response { return &http.Response{StatusCode: code} }
	for _, code := range []int{400, 404, 422, 401, 403, 429} {
		if !isNonRetryableClientResponse(resp(code), nil) {
			t.Fatalf("status %d should be non-retryable", code)
		}
	}
	if isNonRetryableClientResponse(resp(500), nil) {
		t.Fatal("500 should stay retryable")
	}
	if isNonRetryableClientResponse(nil, errTestSentinel) {
		t.Fatal("transport error should stay retryable")
	}
}
