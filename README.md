# db2xlsx

`db2xlsx` is a Go CLI tool that reads one database connection and one or more SQL `SELECT` blocks from a Markdown file, executes them in order, and writes the results to an Excel workbook.

Each SQL result is written to one sheet. The first row contains headers, and the following rows contain query results with basic Excel formatting based on database types.

## Input Format

Use YAML front matter for the database connection and fenced `sql` code blocks for queries.

````markdown
---
database:
  type: postgresql
  host: postgres.example.com
  port: 5432
  database: salesdb
  schema: public
  username: report_user
  password: ${DB_PASSWORD}
  sslmode: require
  connect_timeout_seconds: 30
---

```sql name="Sales"
SELECT * FROM orders
```
````

`name="..."` is used as the Excel sheet name. If omitted, the tool uses `SQL_001`, `SQL_002`, and so on.

Passwords can be written as `${ENV_NAME}` to read from an environment variable on both Windows and Linux.

## Usage

```sh
db2xlsx --input queries.md --output-dir ./reports
```

The output directory is created automatically when it does not exist. The generated file name is based on the execution time:

```text
YYYYMMDD_HHMMSS.xlsx
```

Example:

```text
20260704_153045.xlsx
```

On success, the generated Excel path is printed to stdout. Logs and errors are printed to stderr.

## Options

```text
-i, --input string             Markdown input file path
-o, --output-dir string        Directory where the generated .xlsx file will be written
    --query-timeout duration   Timeout for each SQL query, for example 30s or 5m
    --connect-timeout duration Database connection timeout (default 30s)
    --max-rows int             Maximum rows per SQL result. Zero means unlimited
    --log-level string         debug, info, warn, error (default "info")
    --log-format string        text, json (default "text")
```

## Supported Databases

- PostgreSQL via `github.com/jackc/pgx/v5/stdlib`
- Oracle via `github.com/sijms/go-ora/v2`

The Oracle driver is pure Go, so the basic build does not require Oracle Instant Client.

## Build

Install Go 1.22 or later.

Windows:

```powershell
go mod tidy
go build -o .\bin\db2xlsx.exe .\cmd\db2xlsx
```

Linux:

```sh
go mod tidy
go build -o ./bin/db2xlsx ./cmd/db2xlsx
```

Cross-compile examples:

```sh
GOOS=windows GOARCH=amd64 go build -o ./bin/db2xlsx.exe ./cmd/db2xlsx
GOOS=linux GOARCH=amd64 go build -o ./bin/db2xlsx ./cmd/db2xlsx
```

## Samples

- `samples/oracle.md`
- `samples/postgresql.md`

## Notes

- Only SQL that starts with `SELECT` or `WITH` after leading comments is accepted.
- Non-SELECT SQL is rejected before connecting to the database.
- Excel sheet names are sanitized for Excel restrictions.
- Markdown files are treated as UTF-8 and CRLF/LF line endings are both supported.
