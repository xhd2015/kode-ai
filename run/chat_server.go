package run

import (
	"fmt"

	"github.com/xhd2015/kode-ai/chat/server"
	"github.com/xhd2015/less-gen/flags"
)

const helpChatServer = `chat-server - Start a WebSocket chat server

Usage: kode chat-server [OPTIONS]

Options:
  --listen PORT          port to listen on (default: 8080)
  -v,--verbose           show verbose info
  -h,--help              show this help message

The server exposes a WebSocket endpoint at /stream that supports all events from types.Message.

Examples:
  kode chat-server --listen 8080
  kode chat-server --listen 3000 --verbose
`

// handleChatServer starts a WebSocket chat server
func handleChatServer(args []string) error {
	var verbose bool
	var listen int = 8080

	flagsParser := flags.Bool("-v,--verbose", &verbose).
		Int("--listen", &listen).
		Help("-h,--help", helpChatServer)

	args, err := flagsParser.Parse(args)
	if err != nil {
		return err
	}

	if len(args) > 0 {
		return fmt.Errorf("unexpected arguments: %v", args)
	}

	// Create server options (only server-level configuration)
	serverOpts := server.ServerOptions{
		Verbose: verbose,
	}

	// Start the server
	return server.Start(listen, serverOpts)
}
