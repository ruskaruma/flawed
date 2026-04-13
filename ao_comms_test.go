package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// aoEvent represents a single event in the AO comms protocol.
type aoEvent struct {
	Version int    `json:"v"`
	ID      int    `json:"id"`
	Epoch   int    `json:"epoch"`
	TS      string `json:"ts"`
	Source  string `json:"source"`
	Type    string `json:"type"`
	Message string `json:"message"`
}

func TestAOInboxParsesValidJSONL(t *testing.T) {
	lines := []string{
		`{"v":1,"id":1,"epoch":0,"ts":"2026-04-13T00:00:00Z","source":"orchestrator","type":"instruction","message":"hello"}`,
		`{"v":1,"id":2,"epoch":0,"ts":"2026-04-13T00:00:01Z","source":"orchestrator","type":"instruction","message":"world"}`,
	}

	dir := t.TempDir()
	inbox := filepath.Join(dir, "inbox.jsonl")

	var content []byte
	for _, line := range lines {
		content = append(content, []byte(line+"\n")...)
	}
	if err := os.WriteFile(inbox, content, 0644); err != nil {
		t.Fatalf("writing inbox: %v", err)
	}

	data, err := os.ReadFile(inbox)
	if err != nil {
		t.Fatalf("reading inbox: %v", err)
	}

	// Parse each line as a JSON event
	parsed := 0
	start := 0
	for i := 0; i < len(data); i++ {
		if data[i] == '\n' {
			line := data[start:i]
			start = i + 1
			if len(line) == 0 {
				continue
			}
			var evt aoEvent
			if err := json.Unmarshal(line, &evt); err != nil {
				t.Errorf("line %d: invalid JSON: %v", parsed+1, err)
				continue
			}
			if evt.Version != 1 {
				t.Errorf("line %d: expected v=1, got %d", parsed+1, evt.Version)
			}
			if evt.Source != "orchestrator" {
				t.Errorf("line %d: expected source=orchestrator, got %q", parsed+1, evt.Source)
			}
			if evt.Type != "instruction" {
				t.Errorf("line %d: expected type=instruction, got %q", parsed+1, evt.Type)
			}
			parsed++
		}
	}

	if parsed != len(lines) {
		t.Errorf("expected %d events, parsed %d", len(lines), parsed)
	}
}

func TestAOEventRoundTrip(t *testing.T) {
	evt := aoEvent{
		Version: 1,
		ID:      1,
		Epoch:   0,
		TS:      "2026-04-13T00:00:00Z",
		Source:  "agent",
		Type:    "completion",
		Message: "task done",
	}

	data, err := json.Marshal(evt)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var decoded aoEvent
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if decoded != evt {
		t.Errorf("round-trip mismatch: got %+v, want %+v", decoded, evt)
	}
}
