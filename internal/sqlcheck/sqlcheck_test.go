package sqlcheck

import "testing"

func TestIsSelect(t *testing.T) {
	tests := []struct {
		name string
		sql  string
		want bool
	}{
		{name: "select", sql: "SELECT * FROM users", want: true},
		{name: "with", sql: "WITH x AS (SELECT 1) SELECT * FROM x", want: true},
		{name: "leading comment", sql: "-- comment\nSELECT 1", want: true},
		{name: "update", sql: "UPDATE users SET name = 'x'", want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsSelect(tt.sql); got != tt.want {
				t.Fatalf("IsSelect() = %v, want %v", got, tt.want)
			}
		})
	}
}
