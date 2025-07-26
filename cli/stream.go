package cli

import (
	"bufio"
	"io"
)

type LinesWritterOption func(cfg *linesWritterOption)

type linesWritterOption struct {
	EndCallback func(err error)
}

func WithEndCallback(callback func(err error)) LinesWritterOption {
	return func(cfg *linesWritterOption) {
		cfg.EndCallback = callback
	}
}

func LinesWritter(callback func(line string) bool, opts ...LinesWritterOption) (io.Writer, func()) {
	cfg := linesWritterOption{}
	for _, opt := range opts {
		opt(&cfg)
	}

	stderrReader, stderrWriter := io.Pipe()

	done := make(chan struct{})
	go func() {
		scanner := bufio.NewScanner(stderrReader)
		for scanner.Scan() {
			line := scanner.Text()
			if !callback(line) {
				break
			}
		}
		if cfg.EndCallback != nil {
			if err := scanner.Err(); err != nil {
				cfg.EndCallback(err)
			}
		}
		close(done)
		stderrReader.Close()
	}()
	return stderrWriter, func() {
		stderrWriter.Close()
		<-done
	}
}
