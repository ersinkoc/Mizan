package main

import (
	"os"
	"testing"
)

func TestMainFunctionSuccess(t *testing.T) {
	oldArgs := os.Args
	t.Cleanup(func() { os.Args = oldArgs })
	os.Args = []string{"mizan", "version"}
	main()
}
