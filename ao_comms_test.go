package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// aoMessage represents a single JSONL message from the AO inbox.
type aoMessage struct {
	Version int    `json:"v"`
	ID      int    `json:"id"`
	Epoch   int    `json:"epoch"`
	TS      string `json:"ts"`
	Source  string `json:"source"`
	Type    string `json:"type"`
	Message string `json:"message"`
	Dedup   string `json:"dedup"`
}

func TestAOInboxParse(t *testing.T) {
	// Write a synthetic inbox JSONL file and verify we can round-trip it.
	dir := t.TempDir()
	inboxPath := filepath.Join(dir, "inbox")

	msg := aoMessage{
		Version: 1,
		ID:      42,
		Epoch:   0,
		TS:      "2026-04-13T00:00:00.000Z",
		Source:  "orchestrator",
		Type:    "instruction",
		Message: "hello from the smoke test",
		Dedup:   "test-dedup-1",
	}

	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	data = append(data, '\n')

	if err := os.WriteFile(inboxPath, data, 0644); err != nil {
		t.Fatalf("write inbox: %v", err)
	}

	// Read it back and parse.
	raw, err := os.ReadFile(inboxPath)
	if err != nil {
		t.Fatalf("read inbox: %v", err)
	}

	var got aoMessage
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if got.Type != "instruction" {
		t.Errorf("expected type 'instruction', got %q", got.Type)
	}
	if got.Message != "hello from the smoke test" {
		t.Errorf("unexpected message: %q", got.Message)
	}
	if got.ID != 42 {
		t.Errorf("expected id 42, got %d", got.ID)
	}
}

func TestAOInboxMultiLine(t *testing.T) {
	// Verify we can parse multiple JSONL lines (simulating several messages).
	dir := t.TempDir()
	inboxPath := filepath.Join(dir, "inbox")

	messages := []aoMessage{
		{Version: 1, ID: 1, Type: "instruction", Message: "first"},
		{Version: 1, ID: 2, Type: "instruction", Message: "second"},
		{Version: 1, ID: 3, Type: "status", Message: "third"},
	}

	var fileData []byte
	for _, m := range messages {
		line, err := json.Marshal(m)
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
		fileData = append(fileData, line...)
		fileData = append(fileData, '\n')
	}

	if err := os.WriteFile(inboxPath, fileData, 0644); err != nil {
		t.Fatalf("write inbox: %v", err)
	}

	raw, err := os.ReadFile(inboxPath)
	if err != nil {
		t.Fatalf("read inbox: %v", err)
	}

	dec := json.NewDecoder(strings.NewReader(string(raw)))
	var parsed []aoMessage
	for dec.More() {
		var m aoMessage
		if err := dec.Decode(&m); err != nil {
			t.Fatalf("decode: %v", err)
		}
		parsed = append(parsed, m)
	}

	if len(parsed) != 3 {
		t.Fatalf("expected 3 messages, got %d", len(parsed))
	}
	if parsed[1].Message != "second" {
		t.Errorf("expected 'second', got %q", parsed[1].Message)
	}
}
