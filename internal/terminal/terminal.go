package terminal

import (
	"fmt"
	"io"
	"os"

	"golang.org/x/term"
)

func IsStdinTTY() bool {
	return term.IsTerminal(int(os.Stdin.Fd()))
}

func IsStdoutTerminal() bool {
	return term.IsTerminal(int(os.Stdout.Fd()))
}

func isStdinTTYOld() bool {
	fileInfo, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return (fileInfo.Mode() & os.ModeCharDevice) == os.ModeCharDevice
}

func RequireReadNonTTYStdin() ([]byte, error) {
	if IsStdinTTY() {
		return nil, fmt.Errorf("requires data from stdin")
	}
	bodyBytes, err := io.ReadAll(os.Stdin)
	if err != nil {
		return nil, err
	}
	return bodyBytes, nil
}
