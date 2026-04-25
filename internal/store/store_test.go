package store

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/mizanproxy/mizan/internal/ir"
)

func TestStoreSnapshotsTagsAndRevert(t *testing.T) {
	ctx := context.Background()
	st := New(t.TempDir())
	meta, model, version, err := st.CreateProject(ctx, "edge", "", []ir.Engine{ir.EngineHAProxy})
	if err != nil {
		t.Fatal(err)
	}
	model.Description = "changed"
	version2, err := st.SaveIR(ctx, meta.ID, model, version)
	if err != nil {
		t.Fatal(err)
	}
	snapshots, err := st.ListSnapshots(ctx, meta.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(snapshots) < 2 {
		t.Fatalf("expected snapshots, got %v", snapshots)
	}
	tag, err := st.TagSnapshot(ctx, meta.ID, version2[:12], "release")
	if err != nil {
		t.Fatal(err)
	}
	if tag.Label != "release" {
		t.Fatalf("unexpected tag: %+v", tag)
	}
	reverted, _, err := st.RevertSnapshot(ctx, meta.ID, "release", "")
	if err != nil {
		t.Fatal(err)
	}
	if reverted.Description != "changed" {
		t.Fatalf("unexpected reverted model: %+v", reverted)
	}
}

func TestAuditAppendAndList(t *testing.T) {
	ctx := context.Background()
	st := New(t.TempDir())
	meta, _, _, err := st.CreateProject(ctx, "edge", "", []ir.Engine{ir.EngineHAProxy})
	if err != nil {
		t.Fatal(err)
	}
	if err := st.AppendAudit(ctx, AuditEvent{ProjectID: meta.ID, Actor: "test", Action: "ir.patch", IRSnapshotHash: "abc", Outcome: "success"}); err != nil {
		t.Fatal(err)
	}
	events, err := st.ListAudit(ctx, meta.ID, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 1 {
		t.Fatalf("events=%d", len(events))
	}
	if events[0].Action != "ir.patch" || events[0].Actor != "test" {
		t.Fatalf("unexpected event: %+v", events[0])
	}
}

func TestStoreProjectImportListDeleteAndConflicts(t *testing.T) {
	ctx := context.Background()
	st := New(t.TempDir())
	model := ir.EmptyModel("", "imported", "", []ir.Engine{ir.EngineNginx})
	meta, imported, version, err := st.ImportProject(ctx, "", "", model)
	if err != nil {
		t.Fatal(err)
	}
	if imported.ID != meta.ID || version == "" {
		t.Fatalf("unexpected import result: %+v %+v %q", meta, imported, version)
	}
	projects, err := st.ListProjects(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(projects) != 1 {
		t.Fatalf("projects=%d", len(projects))
	}
	if _, err := st.GetProject(ctx, meta.ID); err != nil {
		t.Fatal(err)
	}
	imported.Description = "changed"
	if _, err := st.SaveIR(ctx, meta.ID, imported, "wrong-version"); err != ErrVersionConflict {
		t.Fatalf("expected conflict, got %v", err)
	}
	if _, err := st.ListSnapshotTags(ctx, meta.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := st.TagSnapshot(ctx, meta.ID, version, ""); err == nil {
		t.Fatal("expected empty tag label error")
	}
	if err := st.DeleteProject(ctx, meta.ID); err != nil {
		t.Fatal(err)
	}
	projects, err = st.ListProjects(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(projects) != 0 {
		t.Fatalf("projects after delete=%d", len(projects))
	}
}

func TestEmptyCollections(t *testing.T) {
	ctx := context.Background()
	st := New(t.TempDir())
	if got := st.Root(); got == "" {
		t.Fatal("root should not be empty")
	}
	if snapshots, err := st.ListSnapshots(ctx, "missing"); err == nil && len(snapshots) != 0 {
		t.Fatalf("unexpected snapshots: %v", snapshots)
	}
	if audit, err := st.ListAudit(ctx, "missing", 10); err != nil || len(audit) != 0 {
		t.Fatalf("audit=%v err=%v", audit, err)
	}
}

func TestDefaultRootEnvAndStoreErrorBranches(t *testing.T) {
	old := os.Getenv("MIZAN_HOME")
	t.Cleanup(func() { _ = os.Setenv("MIZAN_HOME", old) })
	if err := os.Setenv("MIZAN_HOME", "custom-home"); err != nil {
		t.Fatal(err)
	}
	if DefaultRoot() != "custom-home" {
		t.Fatalf("DefaultRoot did not honor env: %q", DefaultRoot())
	}

	ctx := context.Background()
	st := New(t.TempDir())
	model := ir.EmptyModel("missing", "missing", "", []ir.Engine{ir.EngineHAProxy})
	if _, err := st.SaveIR(ctx, "missing", model, ""); err == nil {
		t.Fatal("expected missing project save error")
	}
	if _, _, err := st.GetSnapshot(ctx, "missing", "nope"); err == nil {
		t.Fatal("expected missing snapshot error")
	}
	if _, _, err := st.RevertSnapshot(ctx, "missing", "nope", ""); err == nil {
		t.Fatal("expected missing revert error")
	}
	if err := st.AppendAudit(ctx, AuditEvent{Action: "bad"}); err == nil {
		t.Fatal("expected audit missing project id error")
	}
}

func TestAuditLimitAndDefaults(t *testing.T) {
	ctx := context.Background()
	st := New(t.TempDir())
	meta, _, _, err := st.CreateProject(ctx, "edge", "", []ir.Engine{ir.EngineHAProxy})
	if err != nil {
		t.Fatal(err)
	}
	if err := st.AppendAudit(ctx, AuditEvent{ProjectID: meta.ID, Action: "older", Timestamp: time.Now().Add(-time.Hour)}); err != nil {
		t.Fatal(err)
	}
	if err := st.AppendAudit(ctx, AuditEvent{ProjectID: meta.ID, Action: "newer"}); err != nil {
		t.Fatal(err)
	}
	events, err := st.ListAudit(ctx, meta.ID, 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 1 || events[0].Action != "newer" || events[0].Actor != "local" || events[0].Outcome != "success" {
		t.Fatalf("unexpected limited audit events: %+v", events)
	}
}
