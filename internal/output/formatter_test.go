package output

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"strings"
	"testing"
	"time"
)

func TestParseFormat(t *testing.T) {
	tests := []struct {
		input string
		want  Format
	}{
		{"text", FormatText},
		{"TEXT", FormatText},
		{"Text", FormatText},
		{"json", FormatJSON},
		{"JSON", FormatJSON},
		{"xml", FormatXML},
		{"XML", FormatXML},
		{"yaml", FormatText},
		{"", FormatText},
		{"unknown", FormatText},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := ParseFormat(tt.input)
			if got != tt.want {
				t.Errorf("ParseFormat(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestNewFormatter(t *testing.T) {
	buf := &bytes.Buffer{}
	f := NewFormatter(FormatJSON, buf)
	if f.format != FormatJSON {
		t.Errorf("format = %q, want json", f.format)
	}
	if f.writer != buf {
		t.Error("writer not set correctly")
	}
}

func TestFormatSTUN_Text(t *testing.T) {
	buf := &bytes.Buffer{}
	f := NewFormatter(FormatText, buf)

	result := STUNResult{
		Success:    true,
		Server:     "stun.example.com:3478",
		Duration:   "150ms",
		PublicIP:   "203.0.113.1",
		PublicPort: 54321,
		NATType:    "open_internet",
		LocalIP:    "10.0.0.1",
		Timestamp:  "2026-06-13T12:00:00Z",
	}

	err := f.FormatSTUN(&result)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()

	expectedStrings := []string{
		"=== STUN Test Result ===",
		"STUN",
		"stun.example.com:3478",
		"150ms",
		"SUCCESS",
		"203.0.113.1",
		"54321",
		"10.0.0.1",
		"open_internet",
		"2026-06-13T12:00:00Z",
	}

	for _, s := range expectedStrings {
		if !strings.Contains(output, s) {
			t.Errorf("text output missing %q", s)
		}
	}

	if strings.Count(output, "=== STUN Test Result ===") != 1 {
		t.Error("expected single STUN header")
	}
}

func TestFormatSTUN_TextFailure(t *testing.T) {
	buf := &bytes.Buffer{}
	f := NewFormatter(FormatText, buf)

	result := STUNResult{
		Success:   false,
		Server:    "stun.example.com:3478",
		Duration:  "5s",
		Error:     "connection refused",
		Timestamp: "2026-06-13T12:00:00Z",
	}

	err := f.FormatSTUN(&result)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()

	if !strings.Contains(output, "FAILED") {
		t.Error("text output missing FAILED status")
	}
	if !strings.Contains(output, "connection refused") {
		t.Error("text output missing error message")
	}
}

func TestFormatSTUN_JSON(t *testing.T) {
	buf := &bytes.Buffer{}
	f := NewFormatter(FormatJSON, buf)

	result := STUNResult{
		Success:    true,
		Server:     "stun.example.com:3478",
		Duration:   "150ms",
		PublicIP:   "203.0.113.1",
		PublicPort: 54321,
		NATType:    "open_internet",
		LocalIP:    "10.0.0.1",
		Timestamp:  "2026-06-13T12:00:00Z",
	}

	err := f.FormatSTUN(&result)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var parsed STUNResult
	if err := json.Unmarshal(buf.Bytes(), &parsed); err != nil {
		t.Fatalf("invalid JSON output: %v", err)
	}

	if parsed.Success != result.Success {
		t.Errorf("Success = %v, want %v", parsed.Success, result.Success)
	}
	if parsed.PublicPort != result.PublicPort {
		t.Errorf("PublicPort = %d, want %d", parsed.PublicPort, result.PublicPort)
	}
	if parsed.Server != result.Server {
		t.Errorf("Server = %q, want %q", parsed.Server, result.Server)
	}
}

func TestFormatSTUN_XML(t *testing.T) {
	buf := &bytes.Buffer{}
	f := NewFormatter(FormatXML, buf)

	result := STUNResult{
		Success:    true,
		Server:     "stun.example.com:3478",
		Duration:   "150ms",
		PublicIP:   "203.0.113.1",
		PublicPort: 54321,
		NATType:    "open_internet",
		LocalIP:    "10.0.0.1",
		Timestamp:  "2026-06-13T12:00:00Z",
	}

	err := f.FormatSTUN(&result)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()

	var parsed STUNResult
	if err := xml.Unmarshal([]byte(output), &parsed); err != nil {
		t.Fatalf("invalid XML output: %v\nOutput: %s", err, output)
	}

	if parsed.Success != result.Success {
		t.Errorf("Success = %v, want %v", parsed.Success, result.Success)
	}
	if parsed.Server != result.Server {
		t.Errorf("Server = %q, want %q", parsed.Server, result.Server)
	}
}

func TestFormatTURN_Text(t *testing.T) {
	buf := &bytes.Buffer{}
	f := NewFormatter(FormatText, buf)

	result := TURNResult{
		Success:     true,
		Server:      "turn.example.com:3478",
		Duration:    "250ms",
		RelayedIP:   "203.0.113.10",
		RelayedPort: 49152,
		Lifetime:    "600s",
		LocalIP:     "10.0.0.1",
		Protocol:    "udp",
		Timestamp:   "2026-06-13T12:00:00Z",
	}

	err := f.FormatTURN(&result)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()

	expectedStrings := []string{
		"=== TURN Test Result ===",
		"turn.example.com:3478",
		"250ms",
		"SUCCESS",
		"203.0.113.10",
		"49152",
		"600s",
		"10.0.0.1",
		"udp",
	}

	for _, s := range expectedStrings {
		if !strings.Contains(output, s) {
			t.Errorf("text output missing %q", s)
		}
	}

	if strings.Count(output, "=== TURN Test Result ===") != 1 {
		t.Error("expected single TURN header")
	}
}

func TestFormatTURN_JSON(t *testing.T) {
	buf := &bytes.Buffer{}
	f := NewFormatter(FormatJSON, buf)

	result := TURNResult{
		Success:     true,
		Server:      "turn.example.com:3478",
		Duration:    "250ms",
		RelayedIP:   "203.0.113.10",
		RelayedPort: 49152,
		Lifetime:    "600s",
		LocalIP:     "10.0.0.1",
		Protocol:    "udp",
		Timestamp:   "2026-06-13T12:00:00Z",
	}

	err := f.FormatTURN(&result)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var parsed TURNResult
	if err := json.Unmarshal(buf.Bytes(), &parsed); err != nil {
		t.Fatalf("invalid JSON output: %v", err)
	}

	if parsed.RelayedPort != result.RelayedPort {
		t.Errorf("RelayedPort = %d, want %d", parsed.RelayedPort, result.RelayedPort)
	}
}

func TestFormatTURN_XML(t *testing.T) {
	buf := &bytes.Buffer{}
	f := NewFormatter(FormatXML, buf)

	result := TURNResult{
		Success:     true,
		Server:      "turn.example.com:3478",
		Duration:    "250ms",
		RelayedIP:   "203.0.113.10",
		RelayedPort: 49152,
		Lifetime:    "600s",
		LocalIP:     "10.0.0.1",
		Protocol:    "udp",
		Timestamp:   "2026-06-13T12:00:00Z",
	}

	err := f.FormatTURN(&result)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var parsed TURNResult
	if err := xml.Unmarshal(buf.Bytes(), &parsed); err != nil {
		t.Fatalf("invalid XML output: %v\nOutput: %s", err, buf.String())
	}

	if parsed.Server != result.Server {
		t.Errorf("Server = %q, want %q", parsed.Server, result.Server)
	}
}

func TestFormatBatch_Text(t *testing.T) {
	buf := &bytes.Buffer{}
	f := NewFormatter(FormatText, buf)

	batch := &BatchReport{
		GeneratedAt: time.Now().Format(time.RFC3339),
		STUN: []STUNResult{
			{
				Success:    true,
				Server:     "stun.l.google.com:19302",
				Duration:   "120ms",
				PublicIP:   "203.0.113.1",
				PublicPort: 12345,
				NATType:    "open_internet",
				LocalIP:    "10.0.0.1",
				Timestamp:  "2026-06-13T12:00:00Z",
			},
		},
		TURN: []TURNResult{
			{
				Success:     true,
				Server:      "turn.example.com:3478",
				Duration:    "300ms",
				RelayedIP:   "203.0.113.10",
				RelayedPort: 49152,
				Lifetime:    "600s",
				LocalIP:     "10.0.0.1",
				Protocol:    "udp",
				Timestamp:   "2026-06-13T12:00:00Z",
			},
		},
		Summary: BatchSummary{
			TotalTests: 2,
			Successful: 2,
			Failed:     0,
		},
	}

	err := f.FormatBatch(batch)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()

	expectedStrings := []string{
		"Traverse Batch Report",
		"Total: 2",
		"Successful: 2",
		"Failed: 0",
		"STUN Results",
		"TURN Results",
		"stun.l.google.com:19302",
		"turn.example.com:3478",
	}

	for _, s := range expectedStrings {
		if !strings.Contains(output, s) {
			t.Errorf("batch text output missing %q", s)
		}
	}
}

func TestFormatBatch_JSON(t *testing.T) {
	buf := &bytes.Buffer{}
	f := NewFormatter(FormatJSON, buf)

	batch := &BatchReport{
		GeneratedAt: "2026-06-13T12:00:00Z",
		STUN: []STUNResult{
			{Success: true, Server: "stun.l.google.com:19302", NATType: "open_internet"},
		},
		TURN: []TURNResult{
			{Success: true, Server: "turn.example.com:3478", Protocol: "udp"},
		},
		Summary: BatchSummary{TotalTests: 2, Successful: 2},
	}

	err := f.FormatBatch(batch)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var parsed BatchReport
	if err := json.Unmarshal(buf.Bytes(), &parsed); err != nil {
		t.Fatalf("invalid JSON: %v\nOutput: %s", err, buf.String())
	}

	if parsed.Summary.TotalTests != 2 {
		t.Errorf("TotalTests = %d, want 2", parsed.Summary.TotalTests)
	}
	if len(parsed.STUN) != 1 {
		t.Errorf("len(STUN) = %d, want 1", len(parsed.STUN))
	}
	if len(parsed.TURN) != 1 {
		t.Errorf("len(TURN) = %d, want 1", len(parsed.TURN))
	}
}

func TestFormatBatch_XML(t *testing.T) {
	buf := &bytes.Buffer{}
	f := NewFormatter(FormatXML, buf)

	batch := &BatchReport{
		GeneratedAt: "2026-06-13T12:00:00Z",
		STUN: []STUNResult{
			{Success: true, Server: "stun.l.google.com:19302"},
		},
		TURN: []TURNResult{
			{Success: true, Server: "turn.example.com:3478"},
		},
		Summary: BatchSummary{TotalTests: 2, Successful: 2},
	}

	err := f.FormatBatch(batch)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var parsed BatchReport
	if err := xml.Unmarshal(buf.Bytes(), &parsed); err != nil {
		t.Fatalf("invalid XML: %v\nOutput: %s", err, buf.String())
	}

	if parsed.Summary.TotalTests != 2 {
		t.Errorf("TotalTests = %d, want 2", parsed.Summary.TotalTests)
	}
}

func TestFormatBatch_EmptyResults(t *testing.T) {
	buf := &bytes.Buffer{}
	f := NewFormatter(FormatText, buf)

	batch := &BatchReport{
		GeneratedAt: time.Now().Format(time.RFC3339),
		Summary:     BatchSummary{TotalTests: 0, Successful: 0, Failed: 0},
	}

	err := f.FormatBatch(batch)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "Total: 0") {
		t.Error("batch output missing Total count")
	}

	if strings.Contains(output, "STUN Results") {
		t.Error("should not contain STUN Results when empty")
	}
	if strings.Contains(output, "TURN Results") {
		t.Error("should not contain TURN Results when empty")
	}
}

func TestNewSTUNResult(t *testing.T) {
	r := NewSTUNResult(true, "server", "100ms", "1.2.3.4", "full_cone", "10.0.0.1", "", 12345)

	if !r.Success {
		t.Error("expected success")
	}
	if r.PublicIP != "1.2.3.4" {
		t.Errorf("PublicIP = %q", r.PublicIP)
	}
	if r.PublicPort != 12345 {
		t.Errorf("PublicPort = %d", r.PublicPort)
	}
	if r.Timestamp == "" {
		t.Error("timestamp should not be empty")
	}
}

func TestNewTURNResult(t *testing.T) {
	r := NewTURNResult(true, "server", "200ms", "5.6.7.8", "600s", "10.0.0.1", "udp", "", 49152)

	if !r.Success {
		t.Error("expected success")
	}
	if r.RelayedIP != "5.6.7.8" {
		t.Errorf("RelayedIP = %q", r.RelayedIP)
	}
	if r.RelayedPort != 49152 {
		t.Errorf("RelayedPort = %d", r.RelayedPort)
	}
	if r.Protocol != "udp" {
		t.Errorf("Protocol = %q", r.Protocol)
	}
	if r.Timestamp == "" {
		t.Error("timestamp should not be empty")
	}
}

func TestFormatter_NilWriter(t *testing.T) {
	f := NewFormatter(FormatText, &bytes.Buffer{})
	if f.writer == nil {
		t.Error("writer should not be nil")
	}
}

func TestWriteJSON_Valid(t *testing.T) {
	buf := &bytes.Buffer{}
	f := NewFormatter(FormatJSON, buf)

	err := f.writeJSON(map[string]string{"key": "value"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, `"key"`) || !strings.Contains(output, `"value"`) {
		t.Errorf("unexpected JSON output: %s", output)
	}
}

func TestWriteXML_Valid(t *testing.T) {
	buf := &bytes.Buffer{}
	f := NewFormatter(FormatXML, buf)

	type testStruct struct {
		XMLName xml.Name `xml:"test"`
		Field   string   `xml:"field"`
	}

	err := f.writeXML(&testStruct{Field: "hello"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "<test>") || !strings.Contains(output, "hello") {
		t.Errorf("unexpected XML output: %s", output)
	}
}
