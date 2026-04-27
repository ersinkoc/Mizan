package doctor

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mizanproxy/mizan/internal/ir"
	"github.com/mizanproxy/mizan/internal/store"
)

func TestRunPassAndWarnings(t *testing.T) {
	st := store.New(t.TempDir())
	meta, _, _, err := st.CreateProject(context.Background(), "edge", "", []ir.Engine{ir.EngineHAProxy})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := st.UpsertTarget(context.Background(), meta.ID, store.Target{Name: "edge-01", Host: "localhost", Engine: ir.EngineHAProxy}); err != nil {
		t.Fatal(err)
	}
	report := Run(context.Background(), st, func(name string) (string, error) {
		if name == "ssh" {
			return "/usr/bin/ssh", nil
		}
		return "", os.ErrNotExist
	})
	if report.Status != StatusWarn || report.ProjectCount != 1 || report.TargetCount != 1 {
		t.Fatalf("unexpected report: %+v", report)
	}
	if !hasCheck(report, "native_haproxy", StatusWarn) || !hasCheck(report, "tool_ssh", StatusPass) {
		t.Fatalf("missing expected checks: %+v", report.Checks)
	}
}

func TestRunFailureBranches(t *testing.T) {
	rootFile := filepath.Join(t.TempDir(), "root-file")
	if err := os.WriteFile(rootFile, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	report := Run(context.Background(), store.New(rootFile), nil)
	if report.Status != StatusFail || !hasCheck(report, "data_root", StatusFail) {
		t.Fatalf("expected data root failure: %+v", report)
	}

	st := store.New(t.TempDir())
	meta, _, _, err := st.CreateProject(context.Background(), "edge", "", []ir.Engine{ir.EngineNginx})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(st.Root(), "projects", meta.ID, "config.json"), []byte("{"), 0o600); err != nil {
		t.Fatal(err)
	}
	report = Run(context.Background(), st, func(string) (string, error) { return "/bin/tool", nil })
	if report.Status != StatusFail || !hasCheck(report, "project_integrity", StatusFail) {
		t.Fatalf("expected integrity failure: %+v", report)
	}
}

func TestRunSecretsFailure(t *testing.T) {
	st := store.New(t.TempDir())
	if err := st.Bootstrap(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(st.Root(), "secrets"), []byte("not a dir"), 0o600); err != nil {
		t.Fatal(err)
	}
	report := Run(context.Background(), st, func(string) (string, error) { return "/bin/tool", nil })
	if report.Status != StatusFail || !hasCheck(report, "secrets", StatusFail) {
		t.Fatalf("expected secrets failure: %+v", report)
	}
}

func TestWorst(t *testing.T) {
	if worst(StatusPass, StatusWarn) != StatusWarn || worst(StatusWarn, StatusFail) != StatusFail || worst(StatusPass, StatusPass) != StatusPass {
		t.Fatal("unexpected status ordering")
	}
}

func hasCheck(report Report, name string, status Status) bool {
	for _, check := range report.Checks {
		if check.Name == name && check.Status == status {
			return true
		}
	}
	return false
}

func TestNativeToolUnusedMessage(t *testing.T) {
	var checks []Check
	checkNativeTool(ir.EngineNginx, "nginx", false, func(string) (string, error) {
		return "", os.ErrNotExist
	}, func(name string, status Status, message string) {
		checks = append(checks, Check{Name: name, Status: status, Message: message})
	})
	if len(checks) != 1 || checks[0].Status != StatusPass || !strings.Contains(checks[0].Message, "no nginx projects") {
		t.Fatalf("unexpected native unused check: %+v", checks)
	}
}
