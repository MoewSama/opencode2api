package main

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync/atomic"
	"time"
)

const idLength = 24 // matches opencode: 12 hex (time) + 12 base62 suffix (after prefix_)

var (
	lastTimestamp int64
	counter       uint32
)

type requestIDs struct {
	Session       string
	Request       string
	Project       string
	ParentSession string
}

func deriveRequestIDs(r *http.Request, body map[string]any) requestIDs {
	signal := firstString(
		r.Header.Get("x-opencode-session"),
		r.Header.Get("x-session-affinity"),
		r.Header.Get("X-Session-Id"),
		r.Header.Get("x-session-id"),
		r.Header.Get("conversation-id"),
		stringAt(body, "conversation_id"),
		stringAt(body, "metadata", "session_id"),
	)
	if signal == "" {
		// Using the first user turn keeps a multi-turn conversation stable as its
		// history grows while separating conversations with different beginnings.
		signal = conversationSeed(body)
	}
	if signal == "" {
		signal = stringAt(body, "previous_response_id")
	}
	if signal == "" || signal == `{}` {
		signal = randomID("fallback", 16)
	}
	session := stableID("ses", signal)
	projectSignal := firstString(r.Header.Get("x-opencode-project"), stringAt(body, "metadata", "project_id"))
	if projectSignal == "" {
		projectSignal = "opencode2api:default-project"
	}
	parentSession := firstString(
		r.Header.Get("x-parent-session-id"),
		stringAt(body, "metadata", "parent_session_id"),
	)
	return requestIDs{
		Session:       session,
		Request:       createTimeID("msg", true),
		Project:       createTimeID("prj", true),
		ParentSession: parentSession,
	}
}

// createTimeID generates an opencode-compatible time-ordered ID.
// Matches packages/opencode/src/id/id.ts create(prefix, direction).
func createTimeID(prefix string, ascending bool) string {
	nowMs := time.Now().UnixMilli()

	// Per-millisecond counter (matches opencode's atomic counter pattern)
	lastTs := atomic.LoadInt64(&lastTimestamp)
	var cnt uint32
	if nowMs != lastTs {
		atomic.StoreInt64(&lastTimestamp, nowMs)
		atomic.StoreUint32(&counter, 1)
		cnt = 1
	} else {
		cnt = atomic.AddUint32(&counter, 1)
	}

	// 56-bit value: high 48 bits = timestamp<<12, low 16 bits = counter
	val := (uint64(nowMs) << 12) | uint64(cnt&0xFFFF)

	// Descending direction: invert bits so newer items sort first
	if !ascending {
		val = ^val & 0x00FFFFFFFFFFFFFF // 56-bit mask
	}

	// Top 48 bits as hex (12 chars) — equivalent to (val >> 8)
	timePart := val >> 8

	buf := make([]byte, 6)
	buf[0] = byte(timePart >> 40)
	buf[1] = byte(timePart >> 32)
	buf[2] = byte(timePart >> 24)
	buf[3] = byte(timePart >> 16)
	buf[4] = byte(timePart >> 8)
	buf[5] = byte(timePart)
	hexPart := hex.EncodeToString(buf)

	// Random base62 suffix of exact length
	suffixLen := idLength - 12
	const base62Chars = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
	suffix := make([]byte, suffixLen)
	rand.Read(suffix)
	for i := range suffix {
		suffix[i] = base62Chars[suffix[i]%62]
	}

	return prefix + "_" + hexPart + string(suffix)
}

func conversationSeed(body map[string]any) string {
	if input, ok := body["input"].(string); ok && input != "" {
		return input
	}
	for _, field := range []string{"messages", "input"} {
		for _, raw := range sliceAt(body, field) {
			item, ok := raw.(map[string]any)
			if !ok || stringAt(item, "role") != "user" {
				continue
			}
			encoded, _ := json.Marshal(item["content"])
			if len(encoded) > 0 && string(encoded) != "null" {
				return string(encoded)
			}
		}
	}
	return ""
}

func stableID(prefix, value string) string {
	sum := sha256.Sum256([]byte(prefix + "\x00" + value))
	return prefix + "_" + hex.EncodeToString(sum[:12])
}

func randomID(prefix string, size int) string {
	buf := make([]byte, size)
	if _, err := rand.Read(buf); err != nil {
		panic(fmt.Sprintf("crypto/rand failed: %v", err))
	}
	return prefix + "_" + hex.EncodeToString(buf)
}

func firstString(values ...string) string {
	for _, value := range values {
		if value = strings.TrimSpace(value); value != "" {
			return value
		}
	}
	return ""
}

func opencodeUserAgent() string {
	return "opencode/1.18.31 ai-sdk/provider-utils/4.0.40 runtime/bun/1.3.14"
}
