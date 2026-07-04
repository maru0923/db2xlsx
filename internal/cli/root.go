package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"
	"time"

	"db2xlsx/internal/app"
	"db2xlsx/internal/version"
	"github.com/spf13/cobra"
)

type options struct {
	input         string
	outputDir     string
	queryTimeout  time.Duration
	connectTimeout time.Duration
	maxRows       int
	logLevel      string
	logFormat     string
}

func Execute(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	opts := options{}
	cmd := newRootCommand(ctx, &opts, stdout, stderr)
	cmd.SetArgs(args)
	cmd.SetOut(stdout)
	cmd.SetErr(stderr)

	if err := cmd.Execute(); err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return exitCode(err)
	}
	return exitSuccess
}

func newRootCommand(ctx context.Context, opts *options, stdout, stderr io.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "db2xlsx",
		Short: "Execute SELECT queries from Markdown and export results to Excel.",
		Long:  "db2xlsx reads one database connection and one or more SELECT SQL blocks from Markdown, runs them in order, and writes each result set to a separate Excel sheet.",
		Example: strings.TrimSpace(`
db2xlsx --input queries.md --output-dir ./reports
db2xlsx -i queries.md -o ./reports --query-timeout 5m --max-rows 10000
`),
		Version: fmt.Sprintf("%s (%s, %s)", version.Version, version.Commit, version.Date),
		RunE: func(cmd *cobra.Command, args []string) error {
			if opts.input == "" {
				return app.ValidationError{Message: "--input is required"}
			}
			if opts.outputDir == "" {
				return app.ValidationError{Message: "--output-dir is required"}
			}
			logger, err := newLogger(stderr, opts.logLevel, opts.logFormat)
			if err != nil {
				return err
			}
			svc := app.Service{
				Logger:         logger,
				Now:            time.Now,
				QueryTimeout:   opts.queryTimeout,
				ConnectTimeout: opts.connectTimeout,
				MaxRows:        opts.maxRows,
			}
			result, err := svc.Run(ctx, app.Request{
				InputPath: opts.input,
				OutputDir: opts.outputDir,
			})
			if err != nil {
				return err
			}
			fmt.Fprintln(stdout, result.OutputPath)
			return nil
		},
	}

	cmd.Flags().StringVarP(&opts.input, "input", "i", "", "Markdown input file path")
	cmd.Flags().StringVarP(&opts.outputDir, "output-dir", "o", "", "Directory where the generated .xlsx file will be written")
	cmd.Flags().DurationVar(&opts.queryTimeout, "query-timeout", 0, "Timeout for each SQL query, for example 30s or 5m. Zero means no per-query timeout")
	cmd.Flags().DurationVar(&opts.connectTimeout, "connect-timeout", 30*time.Second, "Database connection timeout")
	cmd.Flags().IntVar(&opts.maxRows, "max-rows", 0, "Maximum rows per SQL result. Zero means unlimited")
	cmd.Flags().StringVar(&opts.logLevel, "log-level", "info", "Log level: debug, info, warn, error")
	cmd.Flags().StringVar(&opts.logFormat, "log-format", "text", "Log format: text, json")
	return cmd
}

func newLogger(w io.Writer, level, format string) (*slog.Logger, error) {
	var slogLevel slog.Level
	switch strings.ToLower(level) {
	case "debug":
		slogLevel = slog.LevelDebug
	case "info":
		slogLevel = slog.LevelInfo
	case "warn":
		slogLevel = slog.LevelWarn
	case "error":
		slogLevel = slog.LevelError
	default:
		return nil, app.ValidationError{Message: "invalid --log-level: " + level}
	}
	opts := &slog.HandlerOptions{Level: slogLevel}
	switch strings.ToLower(format) {
	case "text":
		return slog.New(slog.NewTextHandler(w, opts)), nil
	case "json":
		return slog.New(slog.NewJSONHandler(w, opts)), nil
	default:
		return nil, app.ValidationError{Message: "invalid --log-format: " + format}
	}
}

func exitCode(err error) int {
	if err == nil {
		return exitSuccess
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return exitTimeout
	}
	var validation app.ValidationError
	if errors.As(err, &validation) {
		return exitValidation
	}
	var notFound app.NotFoundError
	if errors.As(err, &notFound) {
		return exitNotFound
	}
	var permission app.PermissionError
	if errors.As(err, &permission) {
		return exitPermission
	}
	var external app.ExternalError
	if errors.As(err, &external) {
		return exitExternal
	}
	if errors.Is(err, os.ErrNotExist) {
		return exitNotFound
	}
	if errors.Is(err, os.ErrPermission) {
		return exitPermission
	}
	return exitGeneral
}
