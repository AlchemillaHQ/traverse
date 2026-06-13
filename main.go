package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/AlchemillaHQ/traverse/internal/output"
	"github.com/AlchemillaHQ/traverse/internal/stun"
	"github.com/AlchemillaHQ/traverse/internal/turn"
	"github.com/AlchemillaHQ/traverse/internal/version"
)

const banner = `
  ╔══════════════════════════════════════════╗
  ║           T R A V E R S E                ║
  ║     STUN / TURN server probe             ║
  ╚══════════════════════════════════════════╝`

type config struct {
	stunServer   string
	turnServer   string
	username     string
	password     string
	protocol     string
	outputFormat string
	timeout      int
	localIP      string
	showVersion  bool
	showHelp     bool
	batch        bool
}

func main() {
	cfg := parseFlags()

	if cfg.showVersion {
		fmt.Printf("traverse v%s  commit=%s  built=%s\n",
			version.Version, version.Commit, version.BuildDate)
		os.Exit(0)
	}

	if len(os.Args) == 1 || cfg.showHelp {
		if cfg.showHelp {
			usage()
			os.Exit(0)
		}
		showWelcome()
		os.Exit(0)
	}

	if err := cfg.validate(); err != nil {
		fmt.Fprintf(os.Stderr, "\033[31m✗\033[0m %s\n\n", err)
		os.Exit(1)
	}

	if err := run(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "\033[31m✗\033[0m %s\n", err)
		os.Exit(1)
	}
}

func parseFlags() *config {
	cfg := &config{}
	fs := flag.NewFlagSet("traverse", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	fs.Usage = func() {
		usage()
		os.Exit(0)
	}

	fs.StringVar(&cfg.stunServer, "stun-server", "", "")
	fs.StringVar(&cfg.stunServer, "s", "", "")

	fs.StringVar(&cfg.turnServer, "turn-server", "", "")
	fs.StringVar(&cfg.turnServer, "t", "", "")

	fs.StringVar(&cfg.username, "username", "", "")
	fs.StringVar(&cfg.username, "u", "", "")

	fs.StringVar(&cfg.password, "password", "", "")
	fs.StringVar(&cfg.password, "p", "", "")

	fs.StringVar(&cfg.protocol, "protocol", "udp", "")
	fs.StringVar(&cfg.protocol, "P", "udp", "")

	fs.StringVar(&cfg.outputFormat, "output", "text", "")
	fs.StringVar(&cfg.outputFormat, "o", "text", "")

	fs.IntVar(&cfg.timeout, "timeout", 5, "")
	fs.IntVar(&cfg.timeout, "T", 5, "")

	fs.StringVar(&cfg.localIP, "local-ip", "", "")
	fs.StringVar(&cfg.localIP, "l", "", "")

	fs.BoolVar(&cfg.showVersion, "version", false, "")
	fs.BoolVar(&cfg.showVersion, "v", false, "")

	fs.BoolVar(&cfg.showHelp, "help", false, "")
	fs.BoolVar(&cfg.showHelp, "h", false, "")

	fs.BoolVar(&cfg.batch, "batch", false, "")
	fs.BoolVar(&cfg.batch, "b", false, "")

	fs.Parse(os.Args[1:])

	cfg.stunServer = ensurePort(cfg.stunServer, "3478")
	cfg.turnServer = ensurePort(cfg.turnServer, "3478")

	return cfg
}

func ensurePort(addr, defaultPort string) string {
	if addr == "" {
		return addr
	}

	if strings.Contains(addr, "[") {
		idx := strings.LastIndex(addr, "]:")
		if idx == -1 {
			return addr + ":" + defaultPort
		}
		return addr
	}

	if !strings.Contains(addr, ":") {
		return addr + ":" + defaultPort
	}
	return addr
}

func showWelcome() {
	fmt.Println(banner)
	fmt.Println()
	fmt.Println("  Usage:  traverse -s <stun-server>       test a STUN server")
	fmt.Println("          traverse -t <turn-server> ...    test a TURN server")
	fmt.Println()
	fmt.Println("  Quick start ─────────────────────────────────────────────")
	fmt.Println("  traverse -s stun.l.google.com:19302")
	fmt.Println("  traverse -s stun.example.com             (default port 3478)")
	fmt.Println("  traverse -s stun.example.com -o json")
	fmt.Println("  traverse -t turn.example.com -u user -p pass")
	fmt.Println()
	fmt.Println("  Run  traverse --help   for the full option reference.")
	fmt.Println()
}

func usage() {
	fmt.Fprintf(os.Stderr, "%s\n\n", banner)
	fmt.Fprintf(os.Stderr, "\033[1mUSAGE\033[0m\n")
	fmt.Fprintf(os.Stderr, "  traverse [options]\n\n")

	fmt.Fprintf(os.Stderr, "\033[1mSTUN OPTIONS\033[0m\n")
	fmt.Fprintf(os.Stderr, "  -s, --stun-server <host[:port]>  STUN server address (default port: 3478)\n\n")

	fmt.Fprintf(os.Stderr, "\033[1mTURN OPTIONS\033[0m\n")
	fmt.Fprintf(os.Stderr, "  -t, --turn-server <host[:port]>  TURN server address (default port: 3478)\n")
	fmt.Fprintf(os.Stderr, "  -u, --username   <string>        TURN username (required)\n")
	fmt.Fprintf(os.Stderr, "  -p, --password   <string>        TURN password (required)\n")
	fmt.Fprintf(os.Stderr, "  -P, --protocol   udp|tcp         transport protocol (default: udp)\n\n")

	fmt.Fprintf(os.Stderr, "\033[1mOUTPUT\033[0m\n")
	fmt.Fprintf(os.Stderr, "  -o, --output  text|json|xml      output format (default: text)\n")
	fmt.Fprintf(os.Stderr, "  -b, --batch                      combined JSON/XML batch report\n\n")

	fmt.Fprintf(os.Stderr, "\033[1mGENERAL\033[0m\n")
	fmt.Fprintf(os.Stderr, "  -T, --timeout  <seconds>         request timeout (default: 5)\n")
	fmt.Fprintf(os.Stderr, "  -l, --local-ip <ip>              bind to specific local IP\n")
	fmt.Fprintf(os.Stderr, "  -v, --version                    print version and exit\n")
	fmt.Fprintf(os.Stderr, "      --help                       show this help\n\n")

	fmt.Fprintf(os.Stderr, "\033[1mEXAMPLES\033[0m\n")
	fmt.Fprintf(os.Stderr, "  traverse -s stun.l.google.com:19302\n")
	fmt.Fprintf(os.Stderr, "  traverse -s stun.example.com -o json\n")
	fmt.Fprintf(os.Stderr, "  traverse -t turn.example.com -u alice -p s3cret\n")
	fmt.Fprintf(os.Stderr, "  traverse -s stun.example.com -t turn.example.com -u alice -p s3cret -o xml -b\n")
}

func (c *config) validate() error {
	if c.stunServer == "" && c.turnServer == "" {
		return fmt.Errorf("at least one of -s/--stun-server or -t/--turn-server is required")
	}

	if c.turnServer != "" {
		if c.username == "" || c.password == "" {
			return fmt.Errorf("-u/--username and -p/--password are required when testing a TURN server")
		}
		p := strings.ToLower(c.protocol)
		if p != "udp" && p != "tcp" {
			return fmt.Errorf("protocol must be 'udp' or 'tcp', got %q", c.protocol)
		}
		c.protocol = p
	}

	f := strings.ToLower(c.outputFormat)
	if f != "text" && f != "json" && f != "xml" {
		return fmt.Errorf("output format must be 'text', 'json', or 'xml', got %q", c.outputFormat)
	}
	c.outputFormat = f

	if c.timeout < 1 {
		return fmt.Errorf("timeout must be at least 1 second")
	}

	return nil
}

func run(cfg *config) error {
	ctx := context.Background()
	timeout := time.Duration(cfg.timeout) * time.Second
	fmtFormat := output.ParseFormat(cfg.outputFormat)
	fmtr := output.NewFormatter(fmtFormat, os.Stdout)

	var stunResults []output.STUNResult
	var turnResults []output.TURNResult
	var stunResult *stun.Result
	var turnResult *turn.Result
	var hasFailure bool

	if cfg.stunServer != "" {
		client := stun.NewClient(cfg.stunServer, timeout, cfg.localIP)
		stunResult = client.Test(ctx)

		sr := output.NewSTUNResult(
			stunResult.Success, stunResult.Server, stunResult.Duration,
			stunResult.PublicIP, stunResult.NATType, stunResult.LocalIP,
			stunResult.Error, stunResult.PublicPort,
		)
		stunResults = append(stunResults, sr)
		if !stunResult.Success {
			hasFailure = true
		}

		if !cfg.batch {
			if err := fmtr.FormatSTUN(&sr); err != nil {
				return fmt.Errorf("format error: %w", err)
			}
		}
	}

	if cfg.turnServer != "" {
		client := turn.NewClient(cfg.turnServer, cfg.username, cfg.password, cfg.protocol, timeout, cfg.localIP)
		turnResult = client.Test(ctx)

		tr := output.NewTURNResult(
			turnResult.Success, turnResult.Server, turnResult.Duration,
			turnResult.RelayedIP, turnResult.Lifetime, turnResult.LocalIP,
			turnResult.Protocol, turnResult.Error, turnResult.RelayedPort,
		)
		turnResults = append(turnResults, tr)
		if !turnResult.Success {
			hasFailure = true
		}

		if !cfg.batch {
			if err := fmtr.FormatTURN(&tr); err != nil {
				return fmt.Errorf("format error: %w", err)
			}
		}
	}

	if cfg.batch {
		successful, failed := 0, 0
		for _, r := range stunResults {
			if r.Success {
				successful++
			} else {
				failed++
			}
		}
		for _, r := range turnResults {
			if r.Success {
				successful++
			} else {
				failed++
			}
		}

		batch := &output.BatchReport{
			GeneratedAt: time.Now().Format(time.RFC3339),
			STUN:        stunResults,
			TURN:        turnResults,
			Summary: output.BatchSummary{
				TotalTests: len(stunResults) + len(turnResults),
				Successful: successful,
				Failed:     failed,
			},
		}

		if err := fmtr.FormatBatch(batch); err != nil {
			return fmt.Errorf("format error: %w", err)
		}
	}

	if hasFailure {
		os.Exit(1)
	}

	return nil
}
