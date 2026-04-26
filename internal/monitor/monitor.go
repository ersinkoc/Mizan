package monitor

import (
	"context"
	"time"

	"github.com/mizanproxy/mizan/internal/ir"
	"github.com/mizanproxy/mizan/internal/store"
)

type Snapshot struct {
	ProjectID   string           `json:"project_id"`
	GeneratedAt string           `json:"generated_at"`
	Summary     Summary          `json:"summary"`
	Targets     []TargetSnapshot `json:"targets"`
}

type Summary struct {
	TotalTargets int `json:"total_targets"`
	Healthy      int `json:"healthy"`
	Warning      int `json:"warning"`
	Unknown      int `json:"unknown"`
	Failed       int `json:"failed"`
}

type TargetSnapshot struct {
	TargetID string    `json:"target_id"`
	Name     string    `json:"name"`
	Host     string    `json:"host"`
	Engine   ir.Engine `json:"engine"`
	Status   string    `json:"status"`
	Message  string    `json:"message"`
}

type Clock func() time.Time

func SnapshotTargets(ctx context.Context, st *store.Store, projectID string, now Clock) (Snapshot, error) {
	targets, err := st.ListTargets(ctx, projectID)
	if err != nil {
		return Snapshot{}, err
	}
	if now == nil {
		now = func() time.Time { return time.Now().UTC() }
	}
	snapshot := Snapshot{
		ProjectID:   projectID,
		GeneratedAt: now().UTC().Format(time.RFC3339),
		Targets:     make([]TargetSnapshot, 0, len(targets.Targets)),
	}
	for _, target := range targets.Targets {
		snapshot.Targets = append(snapshot.Targets, TargetSnapshot{
			TargetID: target.ID,
			Name:     target.Name,
			Host:     target.Host,
			Engine:   target.Engine,
			Status:   "unknown",
			Message:  "runtime collector is not configured",
		})
	}
	snapshot.Summary = summarize(snapshot.Targets)
	return snapshot, nil
}

func summarize(targets []TargetSnapshot) Summary {
	summary := Summary{TotalTargets: len(targets)}
	for _, target := range targets {
		switch target.Status {
		case "healthy":
			summary.Healthy++
		case "warning":
			summary.Warning++
		case "failed":
			summary.Failed++
		default:
			summary.Unknown++
		}
	}
	return summary
}
