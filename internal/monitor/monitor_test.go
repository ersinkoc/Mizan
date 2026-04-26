package monitor

import (
	"testing"
	"time"

	"github.com/mizanproxy/mizan/internal/ir"
	"github.com/mizanproxy/mizan/internal/store"
)

func TestSnapshotTargets(t *testing.T) {
	st := store.New(t.TempDir())
	meta, _, _, err := st.CreateProject(t.Context(), "edge", "", []ir.Engine{ir.EngineHAProxy})
	if err != nil {
		t.Fatal(err)
	}
	target, err := st.UpsertTarget(t.Context(), meta.ID, store.Target{Name: "edge-01", Host: "10.0.0.10", Engine: ir.EngineHAProxy})
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 4, 26, 12, 0, 0, 0, time.UTC)
	snapshot, err := SnapshotTargets(t.Context(), st, meta.ID, func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}
	if snapshot.ProjectID != meta.ID || snapshot.GeneratedAt != now.Format(time.RFC3339) {
		t.Fatalf("unexpected snapshot header: %+v", snapshot)
	}
	if snapshot.Summary.TotalTargets != 1 || snapshot.Summary.Unknown != 1 || len(snapshot.Targets) != 1 {
		t.Fatalf("unexpected summary: %+v targets=%+v", snapshot.Summary, snapshot.Targets)
	}
	if snapshot.Targets[0].TargetID != target.ID || snapshot.Targets[0].Status != "unknown" {
		t.Fatalf("unexpected target snapshot: %+v", snapshot.Targets[0])
	}
}

func TestSnapshotTargetsDefaultsAndErrors(t *testing.T) {
	st := store.New(t.TempDir())
	if _, err := SnapshotTargets(t.Context(), st, "missing", nil); err == nil {
		t.Fatal("expected missing project error")
	}
	meta, _, _, err := st.CreateProject(t.Context(), "edge", "", []ir.Engine{ir.EngineHAProxy})
	if err != nil {
		t.Fatal(err)
	}
	snapshot, err := SnapshotTargets(t.Context(), st, meta.ID, nil)
	if err != nil {
		t.Fatal(err)
	}
	if snapshot.GeneratedAt == "" || snapshot.Summary.TotalTargets != 0 || len(snapshot.Targets) != 0 {
		t.Fatalf("unexpected empty snapshot: %+v", snapshot)
	}
}

func TestSummarize(t *testing.T) {
	summary := summarize([]TargetSnapshot{
		{Status: "healthy"},
		{Status: "warning"},
		{Status: "failed"},
		{Status: "unknown"},
		{Status: ""},
	})
	if summary.TotalTargets != 5 || summary.Healthy != 1 || summary.Warning != 1 || summary.Failed != 1 || summary.Unknown != 2 {
		t.Fatalf("summary=%+v", summary)
	}
}
