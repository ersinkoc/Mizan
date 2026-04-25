package main

import (
	"context"
	"os"

	"github.com/mizanproxy/mizan/internal/cli"
)

func main() {
	if err := cli.Run(context.Background(), os.Args[1:], os.Stdout, os.Stderr); err != nil {
		os.Exit(1)
	}
}
