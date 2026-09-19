// Command limoni-mcp lets an AI agent drive a running Limoni application
// through the Model Context Protocol.
//
// It is a bridge: MCP over stdio on one side, the application's automation
// socket on the other. The agent gets tools — tree, find, click, press_key,
// type_text, wait_for, screen, status — that address widgets by role, label and
// value, the same semantic tree a screen reader reads, instead of scraping
// characters off a pseudo-terminal and clicking coordinates.
//
// Usage:
//
//	limoni-mcp -socket /run/user/1000/myapp.sock
//
// The application has to opt in, and is only able to in a debug build:
//
//	// go run -tags limoni_debug .
//	limoni.Run(draw, limoni.WithAutomation("/run/user/1000/myapp.sock", limoni.AutomationPolicy{
//		AllowInput: true,
//	}))
//
// Registering it with Claude Code:
//
//	claude mcp add limoni -- limoni-mcp -socket /run/user/1000/myapp.sock
//
// # What this does not change
//
// Every limit is enforced by the application, not by this bridge. Secret
// fields are redacted before they leave the process; input, screen text and
// input values are off unless the application's policy turns them on; and the
// socket admits only the user running the application, checked with the
// kernel. The bridge adds no network listener — it speaks to its parent over
// stdio and to the application over a Unix socket, nothing else.
package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"time"
)

// version is reported to MCP clients in serverInfo.
const version = "0.1.0"

func main() {
	if err := run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr); err != nil {
		fmt.Fprintln(os.Stderr, "limoni-mcp:", err)
		os.Exit(1)
	}
}

func run(args []string, stdin io.Reader, stdout, stderr io.Writer) error {
	flags := flag.NewFlagSet("limoni-mcp", flag.ContinueOnError)
	flags.SetOutput(stderr)
	socket := flags.String("socket", os.Getenv("LIMONI_AUTOMATION_SOCKET"),
		"automation socket of the application (default $LIMONI_AUTOMATION_SOCKET)")
	settle := flags.Duration("settle", 500*time.Millisecond,
		"how long click, press_key and type_text wait for the application to redraw")
	showVersion := flags.Bool("version", false, "print the version and exit")
	flags.Usage = func() {
		fmt.Fprintln(stderr, "Usage: limoni-mcp -socket PATH")
		fmt.Fprintln(stderr, "\nServes MCP over stdio and drives the Limoni application listening on PATH.")
		fmt.Fprintln(stderr, "The application must be built with -tags limoni_debug and use limoni.WithAutomation.")
		fmt.Fprintln(stderr)
		flags.PrintDefaults()
	}
	if err := flags.Parse(args); err != nil {
		return err
	}
	if *showVersion {
		fmt.Fprintln(stdout, "limoni-mcp", version)
		return nil
	}
	if *socket == "" {
		flags.Usage()
		return fmt.Errorf("no socket: pass -socket or set LIMONI_AUTOMATION_SOCKET")
	}
	if *settle <= 0 {
		return fmt.Errorf("-settle must be positive")
	}

	// The application does not have to be running yet: the bridge connects on
	// the first tool call, and again after the application restarts. An agent
	// can be configured once and the application started whenever.
	target := &app{socket: *socket, settle: *settle}
	defer target.close()

	// No signal handling: an MCP client ends a stdio server by closing its
	// stdin, which ends serve, and failing that by terminating it. Catching
	// SIGINT here would only leave the process stuck reading stdin.
	server := newMCPServer("limoni", version, serverInstructions, newTools(target))
	return server.serve(context.Background(), stdin, stdout)
}