package cli

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/xhd2015/kode-ai/types"
)

type ChatOptions struct {
	Token          string
	MaxRound       int
	System         string
	ToolDefaultCwd string
	Tools          []string
	ToolCustom     []string
	ToolCustomJson []string

	Record           string             // record file
	RecordData       []byte             // record data to write to tmp file
	UpdateRecordData func([]byte) error // callback to update record data after spl cli returns

	LogChat bool
	Logger  types.Logger
}

type AskOptions struct {
	Token    string
	TraceID  string
	RecordID int64
	Prompt   string
	Model    string
	Logger   types.Logger
}

// Runner defines the interface for command runners
type Runner interface {
	Stream(ctx context.Context, stdout io.Writer) error
	Output(ctx context.Context) (string, error)
}

type RunCLIOptions struct {
	Cli string
	Dir string

	Env []string

	NoCheckUpgrade bool // If true, add SPL_CHECK_UPGRADE=false to environment

	Logger types.Logger
}

type runner struct {
	dir    string
	cli    string
	env    []string
	args   []string
	logger types.Logger
}

func (r *runner) Stream(ctx context.Context, stdout io.Writer) error {
	logger := getLogger(r.logger)

	stderrWriter, cleanup := LinesWritter(func(line string) bool {
		return true
	}, WithEndCallback(func(err error) {
		if err != nil {
			logger.Log(ctx, types.LogType_Error, "error streaming stderr: %v", err)
		}
	}))
	defer cleanup()

	execCmd := r.newCmd(ctx)
	execCmd.Stdout = stdout
	execCmd.Stderr = stderrWriter
	err := execCmd.Run()
	if err != nil {
		return err
	}

	return nil
}

func (r *runner) Output(ctx context.Context) (string, error) {
	logger := getLogger(r.logger)

	execCmd := r.newCmd(ctx)

	// stderr to logger
	stderrReader, stderrWriter := io.Pipe()
	defer stderrReader.Close()
	defer stderrWriter.Close()

	go func() {
		scanner := bufio.NewScanner(stderrReader)
		for scanner.Scan() {
			line := scanner.Text()
			logger.Log(ctx, types.LogType_Error, "callback: %v", line)
		}
		if err := scanner.Err(); err != nil {
			logger.Log(ctx, types.LogType_Error, "reading stderr: %v", err)
		}
	}()

	execCmd.Stderr = stderrWriter

	data, err := execCmd.Output()
	if err != nil {
		return "", err
	}

	return string(data), nil
}

func (r *runner) newCmd(ctx context.Context) *exec.Cmd {
	logger := getLogger(r.logger)
	logger.Log(ctx, types.LogType_Info, "spawn command: %s %s", r.cli, strings.Join(r.args, " "))
	execCmd := exec.CommandContext(ctx, r.cli, r.args...)
	execCmd.Dir = r.dir
	if len(r.env) > 0 {
		execCmd.Env = append(os.Environ(), r.env...)
	}
	return execCmd
}

// recordRunner wraps a Runner to handle record file operations
type recordRunner struct {
	*runner
	tmpRecordFile    string
	updateRecordData func([]byte) error
	cleanupFunc      func() error
}

func (r *recordRunner) Stream(ctx context.Context, stdout io.Writer) error {
	err := r.runner.Stream(ctx, stdout)
	r.handlePostExecution(ctx)
	return err
}

func (r *recordRunner) Output(ctx context.Context) (string, error) {
	output, err := r.runner.Output(ctx)
	if postErr := r.handlePostExecution(ctx); postErr != nil && err == nil {
		err = postErr
	}
	return output, err
}

func (r *recordRunner) handlePostExecution(ctx context.Context) error {
	defer r.cleanup(ctx)

	// If we have a tmp record file and UpdateRecordData callback, read and call it
	if r.tmpRecordFile != "" && r.updateRecordData != nil {
		data, err := os.ReadFile(r.tmpRecordFile)
		if err != nil {
			return fmt.Errorf("failed to read tmp record file: %v", err)
		}

		if err := r.updateRecordData(data); err != nil {
			return fmt.Errorf("UpdateRecordData callback failed: %v", err)
		}
	}

	return nil
}

func (r *recordRunner) cleanup(ctx context.Context) {
	if r.cleanupFunc != nil {
		if err := r.cleanupFunc(); err != nil {
			logger := getLogger(r.runner.logger)
			logger.Log(ctx, types.LogType_Error, "Failed to cleanup: %v", err)
		}
	}
}

type stdErrLogger struct{}

func (l stdErrLogger) Log(ctx context.Context, logType types.LogType, format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, string(logType)+": "+format, args...)
	if !strings.HasSuffix(format, "\n") {
		fmt.Fprintln(os.Stderr)
	}
}

func getLogger(logger types.Logger) types.Logger {
	if logger == nil {
		return stdErrLogger{}
	}
	return logger
}

// RunCommand executes a general spl command with the given arguments
func RunCommand(ctx context.Context, args []string, opts RunCLIOptions) (Runner, error) {
	if len(args) == 0 {
		return nil, fmt.Errorf("requires command arguments")
	}

	// Set default CLI if not specified
	cli := opts.Cli
	if cli == "" {
		cli = "kode"
	}

	env := opts.Env
	if opts.NoCheckUpgrade {
		// Add SPL_CHECK_UPGRADE=false if NoCheckUpgrade is true
		env = append(env, "SPL_CHECK_UPGRADE=false")
	}

	return &runner{
		dir:    opts.Dir,
		cli:    cli,
		args:   args,
		env:    env,
		logger: opts.Logger,
	}, nil
}
