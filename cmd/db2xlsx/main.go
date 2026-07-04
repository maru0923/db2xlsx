package main

import (
	"context"
	"os"

	"db2xlsx/internal/cli"
)

func main() {
	code := cli.Execute(context.Background(), os.Args[1:], os.Stdout, os.Stderr)
	os.Exit(code)
}
