package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"db2xlsx/internal/config"
	"db2xlsx/internal/db"
	"db2xlsx/internal/excel"
	"db2xlsx/internal/sqlcheck"
)

type Service struct {
	Logger         *slog.Logger
	Now            func() time.Time
	QueryTimeout   time.Duration
	ConnectTimeout time.Duration
	MaxRows        int
}

type Request struct {
	InputPath string
	OutputDir string
}

type Result struct {
	OutputPath string
}

func (s Service) Run(ctx context.Context, req Request) (Result, error) {
	now := s.Now
	if now == nil {
		now = time.Now
	}
	logger := s.Logger
	if logger == nil {
		logger = slog.Default()
	}

	inputPath := filepath.Clean(req.InputPath)
	outputDir := filepath.Clean(req.OutputDir)
	content, err := os.ReadFile(inputPath)
	if err != nil {
		return Result{}, mapFileError(err, "read input file "+inputPath)
	}

	doc, err := config.ParseMarkdown(content)
	if err != nil {
		return Result{}, ValidationError{Message: "invalid Markdown input: " + err.Error()}
	}
	if err := config.Validate(doc); err != nil {
		return Result{}, ValidationError{Message: "invalid Markdown input: " + err.Error()}
	}
	for i, query := range doc.Queries {
		if !sqlcheck.IsSelect(query.SQL) {
			return Result{}, ValidationError{Message: fmt.Sprintf("SQL_%03d is not a SELECT query", i+1)}
		}
	}

	if err := ensureOutputDir(outputDir); err != nil {
		return Result{}, err
	}

	connectCtx := ctx
	var cancel context.CancelFunc
	if s.ConnectTimeout > 0 {
		connectCtx, cancel = context.WithTimeout(ctx, s.ConnectTimeout)
		defer cancel()
	}
	logger.Info("connecting database", "type", doc.Database.Type, "host", doc.Database.Host, "port", doc.Database.Port)
	runner, err := db.Open(connectCtx, doc.Database, s.ConnectTimeout)
	if err != nil {
		return Result{}, ExternalError{Message: fmt.Sprintf("failed to connect database type=%s host=%s port=%d", doc.Database.Type, doc.Database.Host, doc.Database.Port), Err: err}
	}
	defer runner.Close()

	sheets := make([]excel.Sheet, 0, len(doc.Queries))
	for i, query := range doc.Queries {
		queryCtx := ctx
		queryCancel := func() {}
		if s.QueryTimeout > 0 {
			queryCtx, queryCancel = context.WithTimeout(ctx, s.QueryTimeout)
		}
		logger.Info("executing SQL", "index", i+1, "name", query.Name)
		queryResult, err := runner.Query(queryCtx, query.SQL, s.MaxRows)
		queryCancel()
		if err != nil {
			return Result{}, ExternalError{Message: fmt.Sprintf("failed to execute SQL_%03d %q", i+1, safeQueryName(query.Name, i+1)), Err: err}
		}
		sheets = append(sheets, excel.Sheet{Name: safeQueryName(query.Name, i+1), Result: queryResult})
	}

	outputPath := filepath.Join(outputDir, now().Format("20060102_150405")+".xlsx")
	if err := excel.Write(outputPath, sheets); err != nil {
		return Result{}, mapFileError(err, "write Excel file "+outputPath)
	}
	logger.Info("wrote Excel file", "path", outputPath)
	return Result{OutputPath: outputPath}, nil
}

func ensureOutputDir(path string) error {
	info, err := os.Stat(path)
	if err == nil {
		if !info.IsDir() {
			return ValidationError{Message: "--output-dir is not a directory: " + path}
		}
		return nil
	}
	if errors.Is(err, os.ErrNotExist) {
		if err := os.MkdirAll(path, 0755); err != nil {
			return mapFileError(err, "create output directory "+path)
		}
		return nil
	}
	return mapFileError(err, "access output directory "+path)
}

func safeQueryName(name string, index int) string {
	if name != "" {
		return name
	}
	return fmt.Sprintf("SQL_%03d", index)
}

func mapFileError(err error, action string) error {
	switch {
	case errors.Is(err, os.ErrNotExist):
		return NotFoundError{Message: action + ": not found"}
	case errors.Is(err, os.ErrPermission):
		return PermissionError{Message: action + ": permission denied"}
	default:
		return err
	}
}
