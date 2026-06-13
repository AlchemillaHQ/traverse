package output

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"strings"
	"time"
)

type Format string

const (
	FormatText Format = "text"
	FormatJSON Format = "json"
	FormatXML  Format = "xml"
)

func ParseFormat(s string) Format {
	switch strings.ToLower(s) {
	case "json":
		return FormatJSON
	case "xml":
		return FormatXML
	case "text":
		return FormatText
	default:
		return FormatText
	}
}

type TestReport interface {
	IsSuccess() bool
	GetError() string
}

type STUNResult struct {
	Success    bool   `json:"success" xml:"success"`
	Server     string `json:"server" xml:"server"`
	Duration   string `json:"duration" xml:"duration"`
	PublicIP   string `json:"public_ip,omitempty" xml:"PublicIP,omitempty"`
	PublicPort int    `json:"public_port,omitempty" xml:"PublicPort,omitempty"`
	NATType    string `json:"nat_type,omitempty" xml:"NATType,omitempty"`
	LocalIP    string `json:"local_ip,omitempty" xml:"LocalIP,omitempty"`
	Error      string `json:"error,omitempty" xml:"Error,omitempty"`
	Timestamp  string `json:"timestamp" xml:"timestamp"`
}

type TURNResult struct {
	Success     bool   `json:"success" xml:"success"`
	Server      string `json:"server" xml:"server"`
	Duration    string `json:"duration" xml:"duration"`
	RelayedIP   string `json:"relayed_ip,omitempty" xml:"RelayedIP,omitempty"`
	RelayedPort int    `json:"relayed_port,omitempty" xml:"RelayedPort,omitempty"`
	Lifetime    string `json:"lifetime,omitempty" xml:"Lifetime,omitempty"`
	LocalIP     string `json:"local_ip,omitempty" xml:"LocalIP,omitempty"`
	Protocol    string `json:"protocol" xml:"protocol"`
	Error       string `json:"error,omitempty" xml:"Error,omitempty"`
	Timestamp   string `json:"timestamp" xml:"timestamp"`
}

type BatchReport struct {
	GeneratedAt string       `json:"generated_at" xml:"generated_at"`
	STUN        []STUNResult `json:"stun_results,omitempty" xml:"STUNResults>STUNResult,omitempty"`
	TURN        []TURNResult `json:"turn_results,omitempty" xml:"TURNResults>TURNResult,omitempty"`
	Summary     BatchSummary `json:"summary" xml:"Summary"`
}

type BatchSummary struct {
	TotalTests int `json:"total_tests" xml:"TotalTests"`
	Successful int `json:"successful" xml:"Successful"`
	Failed     int `json:"failed" xml:"Failed"`
}

type Formatter struct {
	format Format
	writer io.Writer
}

func NewFormatter(format Format, writer io.Writer) *Formatter {
	return &Formatter{
		format: format,
		writer: writer,
	}
}

func (f *Formatter) FormatSTUN(result *STUNResult) error {
	switch f.format {
	case FormatJSON:
		return f.writeJSON(result)
	case FormatXML:
		return f.writeXML(result)
	default:
		return f.writeSTUNText(result)
	}
}

func (f *Formatter) FormatTURN(result *TURNResult) error {
	switch f.format {
	case FormatJSON:
		return f.writeJSON(result)
	case FormatXML:
		return f.writeXML(result)
	default:
		return f.writeTURNText(result)
	}
}

func (f *Formatter) FormatBatch(report *BatchReport) error {
	switch f.format {
	case FormatJSON:
		return f.writeJSON(report)
	case FormatXML:
		return f.writeXML(report)
	default:
		return f.writeBatchText(report)
	}
}

func (f *Formatter) writeJSON(v any) error {
	encoder := json.NewEncoder(f.writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(v)
}

func (f *Formatter) writeXML(v any) error {
	encoder := xml.NewEncoder(f.writer)
	encoder.Indent("", "  ")
	if err := encoder.Encode(v); err != nil {
		return err
	}
	_, err := fmt.Fprintln(f.writer)
	return err
}

func (f *Formatter) writeSTUNText(r *STUNResult) error {
	var sb strings.Builder
	fmt.Fprintf(&sb, "=== STUN Test Result ===\n")
	fmt.Fprintf(&sb, "Timestamp:  %s\n", r.Timestamp)
	fmt.Fprintf(&sb, "Server:     %s\n", r.Server)
	fmt.Fprintf(&sb, "Duration:   %s\n", r.Duration)

	if r.Success {
		fmt.Fprintf(&sb, "Status:     SUCCESS\n")
		fmt.Fprintf(&sb, "Public IP:  %s\n", r.PublicIP)
		fmt.Fprintf(&sb, "Public Port:%d\n", r.PublicPort)
		fmt.Fprintf(&sb, "Local IP:   %s\n", r.LocalIP)
		fmt.Fprintf(&sb, "NAT Type:   %s\n", r.NATType)
	} else {
		fmt.Fprintf(&sb, "Status:     FAILED\n")
		fmt.Fprintf(&sb, "Error:      %s\n", r.Error)
	}
	sb.WriteString("------------------------\n")

	_, err := fmt.Fprint(f.writer, sb.String())
	return err
}

func (f *Formatter) writeTURNText(r *TURNResult) error {
	var sb strings.Builder
	fmt.Fprintf(&sb, "=== TURN Test Result ===\n")
	fmt.Fprintf(&sb, "Timestamp:    %s\n", r.Timestamp)
	fmt.Fprintf(&sb, "Server:       %s\n", r.Server)
	fmt.Fprintf(&sb, "Protocol:     %s\n", r.Protocol)
	fmt.Fprintf(&sb, "Duration:     %s\n", r.Duration)

	if r.Success {
		fmt.Fprintf(&sb, "Status:       SUCCESS\n")
		fmt.Fprintf(&sb, "Relayed IP:   %s\n", r.RelayedIP)
		fmt.Fprintf(&sb, "Relayed Port: %d\n", r.RelayedPort)
		fmt.Fprintf(&sb, "Lifetime:     %s\n", r.Lifetime)
		fmt.Fprintf(&sb, "Local IP:     %s\n", r.LocalIP)
	} else {
		fmt.Fprintf(&sb, "Status:       FAILED\n")
		fmt.Fprintf(&sb, "Error:        %s\n", r.Error)
	}
	sb.WriteString("--------------------------\n")

	_, err := fmt.Fprint(f.writer, sb.String())
	return err
}

func (f *Formatter) writeBatchText(report *BatchReport) error {
	var sb strings.Builder
	fmt.Fprintf(&sb, "=== Traverse Batch Report ===\n")
	fmt.Fprintf(&sb, "Generated: %s\n", report.GeneratedAt)
	fmt.Fprintf(&sb, "Total: %d | Successful: %d | Failed: %d\n\n",
		report.Summary.TotalTests, report.Summary.Successful, report.Summary.Failed)

	if len(report.STUN) > 0 {
		sb.WriteString("--- STUN Results ---\n")
		for i, r := range report.STUN {
			fmt.Fprintf(&sb, "[%d] Server: %s | Duration: %s\n", i+1, r.Server, r.Duration)
			if r.Success {
				fmt.Fprintf(&sb, "    Public: %s:%d | NAT: %s | Local: %s\n",
					r.PublicIP, r.PublicPort, r.NATType, r.LocalIP)
			} else {
				fmt.Fprintf(&sb, "    FAILED: %s\n", r.Error)
			}
		}
		sb.WriteString("\n")
	}

	if len(report.TURN) > 0 {
		sb.WriteString("--- TURN Results ---\n")
		for i, r := range report.TURN {
			fmt.Fprintf(&sb, "[%d] Server: %s | Protocol: %s | Duration: %s\n",
				i+1, r.Server, r.Protocol, r.Duration)
			if r.Success {
				fmt.Fprintf(&sb, "    Relay: %s:%d | Lifetime: %s | Local: %s\n",
					r.RelayedIP, r.RelayedPort, r.Lifetime, r.LocalIP)
			} else {
				fmt.Fprintf(&sb, "    FAILED: %s\n", r.Error)
			}
		}
	}

	sb.WriteString("-----------------------------\n")
	_, err := fmt.Fprint(f.writer, sb.String())
	return err
}

func NewSTUNResult(success bool, server, duration, publicIP, natType, localIP, errMsg string, publicPort int) STUNResult {
	return STUNResult{
		Success:    success,
		Server:     server,
		Duration:   duration,
		PublicIP:   publicIP,
		PublicPort: publicPort,
		NATType:    natType,
		LocalIP:    localIP,
		Error:      errMsg,
		Timestamp:  time.Now().Format(time.RFC3339),
	}
}

func NewTURNResult(success bool, server, duration, relayedIP, lifetime, localIP, protocol, errMsg string, relayedPort int) TURNResult {
	return TURNResult{
		Success:     success,
		Server:      server,
		Duration:    duration,
		RelayedIP:   relayedIP,
		RelayedPort: relayedPort,
		Lifetime:    lifetime,
		LocalIP:     localIP,
		Protocol:    protocol,
		Error:       errMsg,
		Timestamp:   time.Now().Format(time.RFC3339),
	}
}
