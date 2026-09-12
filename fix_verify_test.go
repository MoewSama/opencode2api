package main

import (
	"os"
	"path/filepath"
	"testing"
)

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
