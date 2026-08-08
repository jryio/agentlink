// Package app implements the agentlink command-line interface.
package app

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
)

const (
	exitDrift = 1
	exitUsage = 2
)

// Streams are injectable command I/O and working-directory state.
type Streams struct {
	In  io.Reader
	Out io.Writer
	Err io.Writer
	CWD string
}

// ExitError carries a deliberate process exit code.
type ExitError struct {
	Code int
	Err  error
}

func (e *ExitError) Error() string {
	if e.Err == nil {
		return fmt.Sprintf("exit status %d", e.Code)
	}
	return e.Err.Error()
}

func (e *ExitError) Unwrap() error { return e.Err }

type globalOptions struct {
	config string
	format string
	quiet  bool
}

type application struct {
	version string
	streams Streams
	global  globalOptions
}

// Run executes a command.
func Run(ctx context.Context, args []string, version string, streams Streams) error {
	if streams.In == nil || streams.Out == nil || streams.Err == nil {
		return &ExitError{Code: exitUsage, Err: errors.New("stdin, stdout, and stderr are required")}
	}
	if streams.CWD == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return &ExitError{Code: exitUsage, Err: fmt.Errorf("get working directory: %w", err)}
		}
		streams.CWD = cwd
	}
	global, command, commandArgs, err := parseGlobal(args)
	if err != nil {
		return &ExitError{Code: exitUsage, Err: err}
	}
	app := &application{version: version, streams: streams, global: global}
	switch command {
	case "check", "status", "audit":
		return app.runCheck(ctx, commandArgs)
	case "sync":
		return app.runSync(ctx, commandArgs)
	case "adopt":
		return app.runAdopt(ctx, commandArgs)
	case "guard":
		return app.runGuard(ctx, commandArgs, false)
	case "remind":
		return app.runGuard(ctx, commandArgs, true)
	case "list":
		return app.runList(commandArgs)
	case "validate":
		return app.runValidate(commandArgs)
	case "doctor":
		return app.runDoctor(commandArgs)
	case "schema":
		return app.runSchema(commandArgs)
	case "init":
		return app.runInit(commandArgs)
	case "help":
		return app.printHelp()
	case "version":
		output := printer{writer: app.streams.Out}
		output.println(app.version)
		return output.err
	default:
		return &ExitError{Code: exitUsage, Err: fmt.Errorf("unknown command %q; run `agentlink help`", command)}
	}
}

func parseGlobal(args []string) (globalOptions, string, []string, error) {
	options := globalOptions{format: "human"}
	command := "check"
	for len(args) > 0 {
		arg := args[0]
		switch arg {
		case "--config", "-c":
			if len(args) < 2 {
				return options, "", nil, errors.New("--config requires a path")
			}
			options.config = args[1]
			args = args[2:]
		case "--format":
			if len(args) < 2 {
				return options, "", nil, errors.New("--format requires human or json")
			}
			options.format = args[1]
			args = args[2:]
		case "--json":
			options.format = "json"
			args = args[1:]
		case "--quiet", "-q":
			options.quiet = true
			args = args[1:]
		case "--help", "-h":
			return options, "help", nil, nil
		case "--version", "-v":
			return options, "version", nil, nil
		default:
			command = arg
			args = args[1:]
			if options.format != "human" && options.format != "json" {
				return options, "", nil, fmt.Errorf("--format must be human or json, got %q", options.format)
			}
			return options, command, args, nil
		}
	}
	if options.format != "human" && options.format != "json" {
		return options, "", nil, fmt.Errorf("--format must be human or json, got %q", options.format)
	}
	return options, command, nil, nil
}
