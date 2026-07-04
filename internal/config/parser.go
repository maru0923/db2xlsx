package config

import (
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

type frontMatter struct {
	Database DatabaseConfig `yaml:"database"`
}

var (
	fenceStartRE = regexp.MustCompile("(?i)^```\\s*sql\\b(.*)$")
	nameAttrRE   = regexp.MustCompile(`\bname\s*=\s*("([^"]*)"|'([^']*)'|([^\s]+))`)
	envRefRE     = regexp.MustCompile(`^\$\{([A-Za-z_][A-Za-z0-9_]*)\}$`)
)

func ParseMarkdown(content []byte) (Document, error) {
	text := strings.ReplaceAll(string(content), "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "\n")

	yamlText, body, ok := splitFrontMatter(text)
	if !ok {
		return Document{}, fmt.Errorf("missing YAML front matter")
	}

	var fm frontMatter
	if err := yaml.Unmarshal([]byte(yamlText), &fm); err != nil {
		return Document{}, fmt.Errorf("invalid YAML front matter: %w", err)
	}
	fm.Database.Password = expandEnvRef(fm.Database.Password)
	if fm.Database.ConnectTimeoutSeconds > 0 {
		fm.Database.ConnectTimeout = time.Duration(fm.Database.ConnectTimeoutSeconds) * time.Second
	}

	queries, err := parseSQLBlocks(body)
	if err != nil {
		return Document{}, err
	}
	return Document{Database: fm.Database, Queries: queries}, nil
}

func Validate(doc Document) error {
	db := doc.Database
	dbType := strings.ToLower(strings.TrimSpace(db.Type))
	if dbType != "oracle" && dbType != "postgresql" && dbType != "postgres" {
		return fmt.Errorf("database.type must be oracle or postgresql")
	}
	if strings.TrimSpace(db.Host) == "" {
		return fmt.Errorf("database.host is required")
	}
	if db.Port <= 0 {
		return fmt.Errorf("database.port is required")
	}
	if strings.TrimSpace(db.Username) == "" {
		return fmt.Errorf("database.username is required")
	}
	if strings.TrimSpace(db.Password) == "" {
		return fmt.Errorf("database.password is required")
	}
	switch dbType {
	case "oracle":
		if strings.TrimSpace(db.ServiceName) == "" && strings.TrimSpace(db.SID) == "" {
			return fmt.Errorf("database.service_name or database.sid is required for oracle")
		}
	default:
		if strings.TrimSpace(db.Database) == "" {
			return fmt.Errorf("database.database is required for postgresql")
		}
	}
	if len(doc.Queries) == 0 {
		return fmt.Errorf("at least one SQL code block is required")
	}
	for i, q := range doc.Queries {
		if strings.TrimSpace(q.SQL) == "" {
			return fmt.Errorf("SQL_%03d is empty", i+1)
		}
	}
	return nil
}

func splitFrontMatter(text string) (string, string, bool) {
	if !strings.HasPrefix(text, "---\n") {
		return "", "", false
	}
	rest := text[len("---\n"):]
	end := strings.Index(rest, "\n---\n")
	if end < 0 {
		return "", "", false
	}
	yamlText := rest[:end]
	body := rest[end+len("\n---\n"):]
	return yamlText, body, true
}

func parseSQLBlocks(body string) ([]Query, error) {
	lines := strings.Split(body, "\n")
	queries := make([]Query, 0)
	inSQL := false
	var name string
	var sqlLines []string

	for _, line := range lines {
		if !inSQL {
			matches := fenceStartRE.FindStringSubmatch(line)
			if matches == nil {
				continue
			}
			inSQL = true
			name = parseNameAttr(matches[1])
			sqlLines = nil
			continue
		}

		if strings.HasPrefix(strings.TrimSpace(line), "```") {
			sql := strings.TrimSpace(strings.Join(sqlLines, "\n"))
			queries = append(queries, Query{Name: name, SQL: sql})
			inSQL = false
			name = ""
			sqlLines = nil
			continue
		}
		sqlLines = append(sqlLines, line)
	}
	if inSQL {
		return nil, fmt.Errorf("unterminated SQL code block")
	}
	return queries, nil
}

func parseNameAttr(attrs string) string {
	matches := nameAttrRE.FindStringSubmatch(attrs)
	if matches == nil {
		return ""
	}
	for _, idx := range []int{2, 3, 4} {
		if matches[idx] != "" {
			if unquoted, err := strconv.Unquote(matches[idx]); err == nil {
				return unquoted
			}
			return matches[idx]
		}
	}
	return ""
}

func expandEnvRef(value string) string {
	trimmed := strings.TrimSpace(value)
	matches := envRefRE.FindStringSubmatch(trimmed)
	if matches == nil {
		return value
	}
	return os.Getenv(matches[1])
}
