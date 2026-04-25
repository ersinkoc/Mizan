package validate

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/mizanproxy/mizan/internal/ir"
)

func TestGenerateAndValidateUnknownTarget(t *testing.T) {
	model := ir.EmptyModel("p_1", "edge", "", []ir.Engine{ir.EngineHAProxy})
	if _, err := Generate(model, ir.Engine("caddy")); err == nil {
		t.Fatal("expected unknown generation target error")
	}
	if _, err := Validate(context.Background(), model, ir.Engine("caddy")); err == nil {
		t.Fatal("expected unknown validation target error")
	}
}

func TestValidateSkipsMissingNativeBinary(t *testing.T) {
	model := ir.EmptyModel("p_1", "edge", "", []ir.Engine{ir.EngineHAProxy})
	result, err := Validate(context.Background(), model, ir.EngineHAProxy)
	if err != nil {
		t.Fatal(err)
	}
	if result.Target != ir.EngineHAProxy {
		t.Fatalf("target=%q", result.Target)
	}
	if result.Native.Available && result.Native.Command == "" {
		t.Fatalf("native result missing command: %+v", result.Native)
	}
}

func TestRunNativeSuccessAndFailureWithFakeBinary(t *testing.T) {
	dir := t.TempDir()
	oldPath := os.Getenv("PATH")
	t.Cleanup(func() { _ = os.Setenv("PATH", oldPath) })
	if err := os.Setenv("PATH", dir+string(os.PathListSeparator)+oldPath); err != nil {
		t.Fatal(err)
	}
	writeFakeBinary(t, filepath.Join(dir, "haproxy.bat"), "@echo off\r\necho ok\r\nexit /b 0\r\n")
	ok := runNative(context.Background(), ir.EngineHAProxy, "global\n")
	if !ok.Available || ok.Skipped || ok.ExitCode != 0 || ok.Stderr == "" {
		t.Fatalf("unexpected success native result: %+v", ok)
	}

	writeFakeBinary(t, filepath.Join(dir, "nginx.bat"), "@echo off\r\necho bad\r\nexit /b 7\r\n")
	failed := runNative(context.Background(), ir.EngineNginx, "events {}\n")
	if !failed.Available || failed.Skipped || failed.ExitCode == 0 || failed.Stderr == "" {
		t.Fatalf("unexpected failed native result: %+v", failed)
	}
}

func writeFakeBinary(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o755); err != nil {
		t.Fatal(err)
	}
}
