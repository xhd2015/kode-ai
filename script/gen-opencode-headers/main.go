package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/xhd2015/kode-ai/providers/opencode"
)

func main() {
	dir := flag.String("dir", "", "directory to generate OpenCode headers for; defaults to current directory")
	oneLine := flag.Bool("one-line", false, "print all headers on one line")
	jsonOutput := flag.Bool("json", false, "print headers as JSON")
	flag.Parse()

	headers, err := opencode.GenerateHeaders(opencode.HeaderOptions{
		Dir: *dir,
	})
	if err != nil {
		exitf("generate headers: %v", err)
	}

	if *jsonOutput {
		var output struct {
			Headers []opencode.Header `json:"headers"`
		}
		output.Headers = headers
		encoder := json.NewEncoder(os.Stdout)
		if err := encoder.Encode(output); err != nil {
			exitf("encode JSON: %v", err)
		}
		return
	}

	if *oneLine {
		for i, header := range headers {
			if i > 0 {
				fmt.Print(" ")
			}
			fmt.Printf("-H %s", shellQuote(header.Name+": "+header.Value))
		}
		fmt.Println()
		return
	}

	for i, header := range headers {
		suffix := ""
		if i < len(headers)-1 {
			suffix = " \\"
		}
		fmt.Printf("-H %s%s\n", shellQuote(header.Name+": "+header.Value), suffix)
	}
}

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\\''") + "'"
}

func exitf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
