package store

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/mizanproxy/mizan/internal/ir"
)

func TestApprovalRequests(t *testing.T) {
	st := New(t.TempDir())
	meta, _, version, err := st.CreateProject(t.Context(), "edge", "", []ir.Engine{ir.EngineHAProxy})
	if err != nil {
		t.Fatal(err)
	}
	request, err := st.CreateApprovalRequest(t.Context(), meta.ID, ApprovalRequest{
		ClusterID:         "cluster-a",
		SnapshotHash:      version,
		Batch:             1,
		RequiredApprovals: 2,
		Approvals: []Approval{
			{Actor: " alice "},
			{Actor: "ALICE", ApprovedAt: time.Now().UTC()},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if request.ID == "" || request.ProjectID != meta.ID || request.Status != ApprovalStatusPending || len(request.Approvals) != 1 {
		t.Fatalf("unexpected approval request: %+v", request)
	}
	request, err = st.ApproveRequest(t.Context(), meta.ID, request.ID, "bob")
	if err != nil {
		t.Fatal(err)
	}
	if request.Status != ApprovalStatusApproved || len(request.Approvals) != 2 {
		t.Fatalf("approval request was not approved: %+v", request)
	}
	request, err = st.ApproveRequest(t.Context(), meta.ID, request.ID, "BOB")
	if err != nil {
		t.Fatal(err)
	}
	if len(request.Approvals) != 2 {
		t.Fatalf("duplicate approval should not be counted: %+v", request.Approvals)
	}
	got, err := st.GetApprovalRequest(t.Context(), meta.ID, request.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.ApprovedActors()[0] != "alice" || got.ApprovedActors()[1] != "bob" {
		t.Fatalf("unexpected approved actors: %v", got.ApprovedActors())
	}
	all, err := st.ListApprovalRequests(t.Context(), meta.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 1 || all[0].ID != request.ID {
		t.Fatalf("unexpected approval list: %+v", all)
	}
}

func TestApprovalRequestErrors(t *testing.T) {
	st := New(t.TempDir())
	meta, _, version, err := st.CreateProject(t.Context(), "edge", "", []ir.Engine{ir.EngineHAProxy})
	if err != nil {
		t.Fatal(err)
	}
	for _, request := range []ApprovalRequest{
		{},
		{TargetID: "target-a", ClusterID: "cluster-a", SnapshotHash: version},
		{TargetID: "target-a"},
		{TargetID: "target-a", SnapshotHash: version, Batch: -1},
		{TargetID: "target-a", SnapshotHash: version, RequiredApprovals: -1},
	} {
		if _, err := st.CreateApprovalRequest(t.Context(), meta.ID, request); err == nil {
			t.Fatalf("expected create approval error for %+v", request)
		}
	}
	if _, err := st.ListApprovalRequests(t.Context(), "missing"); err == nil {
		t.Fatal("expected missing project approval list error")
	}
	if _, err := st.GetApprovalRequest(t.Context(), meta.ID, "missing"); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected missing approval error, got %v", err)
	}
	if _, err := st.ApproveRequest(t.Context(), meta.ID, "missing", "alice"); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected missing approval approve error, got %v", err)
	}
	request, err := st.CreateApprovalRequest(t.Context(), meta.ID, ApprovalRequest{TargetID: "target-a", SnapshotHash: version})
	if err != nil {
		t.Fatal(err)
	}
	if request.Status != ApprovalStatusApproved {
		t.Fatalf("zero-required approval request should be approved: %+v", request)
	}
	if _, err := st.ApproveRequest(t.Context(), meta.ID, request.ID, " "); err == nil {
		t.Fatal("expected blank actor approval error")
	}
	originalRenameFile := renameFile
	renameFile = func(string, string) error {
		return errors.New("rename failed")
	}
	if _, err := st.CreateApprovalRequest(t.Context(), meta.ID, ApprovalRequest{TargetID: "target-write", SnapshotHash: version}); err == nil {
		t.Fatal("expected create approval write error")
	}
	if _, err := st.ApproveRequest(t.Context(), meta.ID, request.ID, "alice"); err == nil {
		t.Fatal("expected approve request write error")
	}
	renameFile = originalRenameFile
	if err := os.WriteFile(st.approvalsPath(meta.ID), []byte("{"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := st.ListApprovalRequests(t.Context(), meta.ID); err == nil {
		t.Fatal("expected corrupt approvals file error")
	}
	if err := os.WriteFile(filepath.Join(st.projectDir(meta.ID), "approvals.json"), []byte("null"), 0o600); err != nil {
		t.Fatal(err)
	}
	all, err := st.ListApprovalRequests(t.Context(), meta.ID)
	if err != nil {
		t.Fatal(err)
	}
	if all == nil || len(all) != 0 {
		t.Fatalf("expected nil approvals normalized to empty slice: %+v", all)
	}
}
