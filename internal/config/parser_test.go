package config

import "testing"

func TestParseMarkdown(t *testing.T) {
	t.Setenv("DB_PASSWORD", "secret")
	content := []byte("---\r\n" +
		"database:\r\n" +
		"  type: postgresql\r\n" +
		"  host: localhost\r\n" +
		"  port: 5432\r\n" +
		"  database: appdb\r\n" +
		"  username: app\r\n" +
		"  password: ${DB_PASSWORD}\r\n" +
		"---\r\n" +
		"\r\n" +
		"```sql name=\"Users\"\r\n" +
		"SELECT id, name FROM users\r\n" +
		"```\r\n")

	doc, err := ParseMarkdown(content)
	if err != nil {
		t.Fatalf("ParseMarkdown returned error: %v", err)
	}
	if doc.Database.Password != "secret" {
		t.Fatalf("password = %q, want %q", doc.Database.Password, "secret")
	}
	if len(doc.Queries) != 1 {
		t.Fatalf("queries = %d, want 1", len(doc.Queries))
	}
	if doc.Queries[0].Name != "Users" {
		t.Fatalf("query name = %q, want Users", doc.Queries[0].Name)
	}
}
