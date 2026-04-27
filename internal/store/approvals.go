package store

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const (
	ApprovalStatusPending  = "pending"
	ApprovalStatusApproved = "approved"
)

type Approval struct {
	Actor      string    `json:"actor"`
	ApprovedAt time.Time `json:"approved_at"`
}

type ApprovalRequest struct {
	ID                string     `json:"id"`
	ProjectID         string     `json:"project_id"`
	TargetID          string     `json:"target_id,omitempty"`
	ClusterID         string     `json:"cluster_id,omitempty"`
	SnapshotHash      string     `json:"snapshot_hash"`
	Batch             int        `json:"batch,omitempty"`
	RequiredApprovals int        `json:"required_approvals"`
	Approvals         []Approval `json:"approvals"`
	Status            string     `json:"status"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
}

func (s *Store) ListApprovalRequests(ctx context.Context, projectID string) ([]ApprovalRequest, error) {
	requests, err := s.readApprovalRequests(projectID)
	if err != nil {
		return nil, err
	}
	sort.Slice(requests, func(i, j int) bool {
		if requests[i].UpdatedAt.Equal(requests[j].UpdatedAt) {
			return requests[i].CreatedAt.After(requests[j].CreatedAt)
		}
		return requests[i].UpdatedAt.After(requests[j].UpdatedAt)
	})
	return requests, nil
}

func (s *Store) GetApprovalRequest(ctx context.Context, projectID, requestID string) (ApprovalRequest, error) {
	requests, err := s.readApprovalRequests(projectID)
	if err != nil {
		return ApprovalRequest{}, err
	}
	for _, request := range requests {
		if request.ID == requestID {
			return request, nil
		}
	}
	return ApprovalRequest{}, os.ErrNotExist
}

func (s *Store) CreateApprovalRequest(ctx context.Context, projectID string, request ApprovalRequest) (ApprovalRequest, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := validateApprovalRequest(request); err != nil {
		return ApprovalRequest{}, err
	}
	requests, err := s.readApprovalRequests(projectID)
	if err != nil {
		return ApprovalRequest{}, err
	}
	now := time.Now().UTC()
	if request.ID == "" {
		request.ID = newID()
	}
	request.ProjectID = projectID
	if request.CreatedAt.IsZero() {
		request.CreatedAt = now
	}
	request.UpdatedAt = now
	request.Approvals = normalizeApprovalRecords(request.Approvals, now)
	request.Status = approvalStatus(request.RequiredApprovals, len(request.Approvals))
	requests = append(requests, request)
	if err := writeJSON(s.approvalsPath(projectID), requests); err != nil {
		return ApprovalRequest{}, err
	}
	return request, nil
}

func (s *Store) ApproveRequest(ctx context.Context, projectID, requestID, actor string) (ApprovalRequest, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	actor = strings.TrimSpace(actor)
	if actor == "" {
		return ApprovalRequest{}, errors.New("actor is required")
	}
	requests, err := s.readApprovalRequests(projectID)
	if err != nil {
		return ApprovalRequest{}, err
	}
	now := time.Now().UTC()
	for i := range requests {
		if requests[i].ID != requestID {
			continue
		}
		if !hasApprovalActor(requests[i].Approvals, actor) {
			requests[i].Approvals = append(requests[i].Approvals, Approval{Actor: actor, ApprovedAt: now})
		}
		requests[i].UpdatedAt = now
		requests[i].Status = approvalStatus(requests[i].RequiredApprovals, len(requests[i].Approvals))
		if err := writeJSON(s.approvalsPath(projectID), requests); err != nil {
			return ApprovalRequest{}, err
		}
		return requests[i], nil
	}
	return ApprovalRequest{}, os.ErrNotExist
}

func (s *Store) readApprovalRequests(projectID string) ([]ApprovalRequest, error) {
	var requests []ApprovalRequest
	if err := readJSON(s.approvalsPath(projectID), &requests); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			if _, statErr := statPath(s.projectDir(projectID)); statErr != nil {
				return nil, statErr
			}
			return []ApprovalRequest{}, nil
		}
		return nil, err
	}
	if requests == nil {
		return []ApprovalRequest{}, nil
	}
	return requests, nil
}

func validateApprovalRequest(request ApprovalRequest) error {
	if request.TargetID == "" && request.ClusterID == "" {
		return errors.New("target_id or cluster_id is required")
	}
	if request.TargetID != "" && request.ClusterID != "" {
		return errors.New("exactly one of target_id or cluster_id is required")
	}
	if request.SnapshotHash == "" {
		return errors.New("snapshot_hash is required")
	}
	if request.Batch < 0 {
		return errors.New("batch must be non-negative")
	}
	if request.RequiredApprovals < 0 {
		return errors.New("required approvals must be non-negative")
	}
	return nil
}

func normalizeApprovalRecords(items []Approval, now time.Time) []Approval {
	seen := map[string]bool{}
	approvals := []Approval{}
	for _, item := range items {
		actor := strings.TrimSpace(item.Actor)
		if actor == "" {
			continue
		}
		key := strings.ToLower(actor)
		if seen[key] {
			continue
		}
		seen[key] = true
		if item.ApprovedAt.IsZero() {
			item.ApprovedAt = now
		}
		item.Actor = actor
		approvals = append(approvals, item)
	}
	return approvals
}

func hasApprovalActor(approvals []Approval, actor string) bool {
	for _, approval := range approvals {
		if strings.EqualFold(approval.Actor, actor) {
			return true
		}
	}
	return false
}

func approvalStatus(requiredApprovals, approvals int) string {
	if approvals >= requiredApprovals {
		return ApprovalStatusApproved
	}
	return ApprovalStatusPending
}

func (request ApprovalRequest) ApprovedActors() []string {
	actors := make([]string, 0, len(request.Approvals))
	for _, approval := range request.Approvals {
		actors = append(actors, approval.Actor)
	}
	return actors
}

func (s *Store) approvalsPath(id string) string {
	return filepath.Join(s.projectDir(id), "approvals.json")
}
