package db

import (
	"context"
	"database/sql"
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"
	"time"

	"db2xlsx/internal/config"
	_ "github.com/jackc/pgx/v5/stdlib"
	_ "github.com/sijms/go-ora/v2"
)

type Runner struct {
	DB *sql.DB
}

type Result struct {
	Columns []Column
	Rows    [][]any
}

type Column struct {
	Name             string
	DatabaseTypeName string
}

func Open(ctx context.Context, cfg config.DatabaseConfig, fallbackConnectTimeout time.Duration) (*Runner, error) {
	driver, dsn, err := buildDSN(cfg, fallbackConnectTimeout)
	if err != nil {
		return nil, err
	}
	db, err := sql.Open(driver, dsn)
	if err != nil {
		return nil, err
	}
	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, err
	}
	return &Runner{DB: db}, nil
}

func (r *Runner) Close() error {
	if r == nil || r.DB == nil {
		return nil
	}
	return r.DB.Close()
}

func (r *Runner) Query(ctx context.Context, sqlText string, maxRows int) (Result, error) {
	rows, err := r.DB.QueryContext(ctx, sqlText)
	if err != nil {
		return Result{}, err
	}
	defer rows.Close()

	names, err := rows.Columns()
	if err != nil {
		return Result{}, err
	}
	types, err := rows.ColumnTypes()
	if err != nil {
		return Result{}, err
	}
	columns := make([]Column, len(names))
	for i, name := range names {
		columns[i] = Column{Name: name}
		if i < len(types) {
			columns[i].DatabaseTypeName = types[i].DatabaseTypeName()
		}
	}

	var resultRows [][]any
	for rows.Next() {
		if maxRows > 0 && len(resultRows) >= maxRows {
			return Result{}, fmt.Errorf("result row count exceeds --max-rows %d", maxRows)
		}
		values := make([]any, len(names))
		dest := make([]any, len(names))
		for i := range values {
			dest[i] = &values[i]
		}
		if err := rows.Scan(dest...); err != nil {
			return Result{}, err
		}
		for i, value := range values {
			if b, ok := value.([]byte); ok {
				values[i] = string(b)
			}
		}
		resultRows = append(resultRows, values)
	}
	if err := rows.Err(); err != nil {
		return Result{}, err
	}
	return Result{Columns: columns, Rows: resultRows}, nil
}

func buildDSN(cfg config.DatabaseConfig, fallbackConnectTimeout time.Duration) (string, string, error) {
	dbType := strings.ToLower(strings.TrimSpace(cfg.Type))
	timeout := cfg.ConnectTimeout
	if timeout == 0 {
		timeout = fallbackConnectTimeout
	}
	switch dbType {
	case "postgres", "postgresql":
		u := &url.URL{
			Scheme: "postgres",
			User:   url.UserPassword(cfg.Username, cfg.Password),
			Host:   net.JoinHostPort(cfg.Host, strconv.Itoa(cfg.Port)),
			Path:   "/" + cfg.Database,
		}
		q := u.Query()
		if cfg.SSLMode != "" {
			q.Set("sslmode", cfg.SSLMode)
		} else {
			q.Set("sslmode", "prefer")
		}
		if timeout > 0 {
			q.Set("connect_timeout", strconv.Itoa(int(timeout.Seconds())))
		}
		if cfg.Schema != "" {
			q.Set("search_path", cfg.Schema)
		}
		u.RawQuery = q.Encode()
		return "pgx", u.String(), nil
	case "oracle":
		service := cfg.ServiceName
		if service == "" {
			service = cfg.SID
		}
		u := &url.URL{
			Scheme: "oracle",
			User:   url.UserPassword(cfg.Username, cfg.Password),
			Host:   net.JoinHostPort(cfg.Host, strconv.Itoa(cfg.Port)),
			Path:   "/" + service,
		}
		q := u.Query()
		if timeout > 0 {
			q.Set("CONNECTION TIMEOUT", strconv.Itoa(int(timeout.Seconds())))
		}
		u.RawQuery = q.Encode()
		return "oracle", u.String(), nil
	default:
		return "", "", fmt.Errorf("unsupported database type: %s", cfg.Type)
	}
}
