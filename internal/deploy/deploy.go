package deploy

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/mizanproxy/mizan/internal/ir"
	"github.com/mizanproxy/mizan/internal/secrets"
	"github.com/mizanproxy/mizan/internal/store"
	"github.com/mizanproxy/mizan/internal/validate"
)

type Request struct {
	ProjectID           string
	TargetID            string
	ClusterID           string
	DryRun              bool
	ConfirmSnapshotHash string
	Batch               int
	ApprovedBy          []string
}

type Result struct {
	ProjectID         string        `json:"project_id"`
	TargetID          string        `json:"target_id,omitempty"`
	ClusterID         string        `json:"cluster_id,omitempty"`
	SnapshotHash      string        `json:"snapshot_hash"`
	DryRun            bool          `json:"dry_run"`
	Batch             int           `json:"batch,omitempty"`
	RequiredApprovals int           `json:"required_approvals,omitempty"`
	ApprovedBy        []string      `json:"approved_by,omitempty"`
	Rollback          RollbackStats `json:"rollback"`
	Cleanup           CleanupStats  `json:"cleanup"`
	Status            string        `json:"status"`
	StartedAt         string        `json:"started_at"`
	FinishedAt        string        `json:"finished_at"`
	Steps             []Step        `json:"steps"`
}

type RollbackStats struct {
	Planned   int `json:"planned"`
	Attempted int `json:"attempted"`
	Succeeded int `json:"succeeded"`
	Failed    int `json:"failed"`
}

type CleanupStats struct {
	Planned   int `json:"planned"`
	Attempted int `json:"attempted"`
	Succeeded int `json:"succeeded"`
	Failed    int `json:"failed"`
}

type Step struct {
	TargetID   string    `json:"target_id"`
	TargetName string    `json:"target_name"`
	Engine     ir.Engine `json:"engine"`
	Stage      string    `json:"stage"`
	Status     string    `json:"status"`
	Command    string    `json:"command,omitempty"`
	Message    string    `json:"message,omitempty"`
	Credential string    `json:"credential_source,omitempty"`
	Batch      int       `json:"batch"`
}

type ProbeResult struct {
	TargetID   string `json:"target_id"`
	TargetName string `json:"target_name"`
	URL        string `json:"url"`
	Status     string `json:"status"`
	Message    string `json:"message,omitempty"`
	CheckedAt  string `json:"checked_at"`
}

type Runner func(context.Context, store.Target, secrets.Secret, string, string) (string, error)
type Prober func(context.Context, string) error
type CredentialProvider func(context.Context, store.Target) (secrets.Secret, error)

type Deployer struct {
	Runner      Runner
	Prober      Prober
	Credentials CredentialProvider
	Now         func() time.Time
}

var (
	createTempKeyFile = func() (string, error) {
		f, err := os.CreateTemp("", "mizan-ssh-key-*")
		if err != nil {
			return "", err
		}
		name := f.Name()
		_ = f.Close()
		return name, nil
	}
	writeKeyFile  = os.WriteFile
	chmodKeyFile  = os.Chmod
	removeKeyFile = os.Remove
)

func New() Deployer {
	return Deployer{Runner: sshRunner, Prober: httpProbe, Now: func() time.Time { return time.Now().UTC() }}
}

func (d Deployer) Run(ctx context.Context, st *store.Store, req Request) (Result, error) {
	if req.ProjectID == "" {
		return Result{}, errors.New("project_id is required")
	}
	if req.TargetID == "" && req.ClusterID == "" {
		return Result{}, errors.New("target_id or cluster_id is required")
	}
	if req.TargetID != "" && req.ClusterID != "" {
		return Result{}, errors.New("exactly one of target_id or cluster_id is required")
	}
	if req.Batch < 0 {
		return Result{}, errors.New("batch must be non-negative")
	}
	model, snapshot, err := st.GetIR(ctx, req.ProjectID)
	if err != nil {
		return Result{}, err
	}
	if !req.DryRun && req.ConfirmSnapshotHash != snapshot {
		return Result{}, errors.New("execute requires confirm_snapshot_hash matching the current project snapshot")
	}
	targetsFile, err := st.ListTargets(ctx, req.ProjectID)
	if err != nil {
		return Result{}, err
	}
	selected, parallelism, gate, requiredApprovals, err := selectTargets(targetsFile, req)
	if err != nil {
		return Result{}, err
	}
	approvedBy, err := normalizeApprovals(req.ApprovedBy)
	if err != nil {
		return Result{}, err
	}
	if !req.DryRun && requiredApprovals > 0 && len(approvedBy) < requiredApprovals {
		return Result{}, fmt.Errorf("execute requires %d distinct approval(s); got %d", requiredApprovals, len(approvedBy))
	}
	selected, err = filterBatch(selected, parallelism, req.Batch)
	if err != nil {
		return Result{}, err
	}
	if d.Runner == nil {
		d.Runner = sshRunner
	}
	if d.Prober == nil {
		d.Prober = httpProbe
	}
	if d.Now == nil {
		d.Now = func() time.Time { return time.Now().UTC() }
	}

	started := d.Now()
	result := Result{
		ProjectID:         req.ProjectID,
		TargetID:          req.TargetID,
		ClusterID:         req.ClusterID,
		SnapshotHash:      snapshot,
		DryRun:            req.DryRun,
		Batch:             req.Batch,
		RequiredApprovals: requiredApprovals,
		ApprovedBy:        approvedBy,
		Status:            "success",
		StartedAt:         started.Format(time.RFC3339),
	}
	for index, target := range selected {
		batch := index/parallelism + 1
		if req.Batch > 0 {
			batch = req.Batch
		}
		credential := secrets.Secret{}
		if !req.DryRun && d.Credentials != nil {
			var err error
			credential, err = d.Credentials(ctx, target)
			if err != nil {
				steps := []Step{credentialFailureStep(target, batch, err)}
				result.Steps = append(result.Steps, steps...)
				result.Status = "failed"
				if gate {
					break
				}
				continue
			}
		}
		steps := d.runTarget(ctx, model, req.ProjectID, target, credential, batch, req.DryRun)
		result.Steps = append(result.Steps, steps...)
		if hasFailed(steps) {
			result.Status = "failed"
			if gate {
				break
			}
		}
	}
	result.Rollback = RollbackSummary(result.Steps)
	result.Cleanup = CleanupSummary(result.Steps)
	result.FinishedAt = d.Now().Format(time.RFC3339)
	return result, nil
}

func filterBatch(targets []store.Target, parallelism int, batch int) ([]store.Target, error) {
	if batch == 0 {
		return targets, nil
	}
	if parallelism <= 0 {
		parallelism = 1
	}
	start := (batch - 1) * parallelism
	if start >= len(targets) {
		return nil, fmt.Errorf("batch %d has no targets", batch)
	}
	end := start + parallelism
	if end > len(targets) {
		end = len(targets)
	}
	return targets[start:end], nil
}

func normalizeApprovals(items []string) ([]string, error) {
	seen := map[string]bool{}
	approvals := []string{}
	for _, item := range items {
		for _, part := range strings.Split(item, ",") {
			approval := strings.TrimSpace(part)
			if approval == "" {
				continue
			}
			key := strings.ToLower(approval)
			if seen[key] {
				continue
			}
			seen[key] = true
			approvals = append(approvals, approval)
		}
	}
	return approvals, nil
}

func (d Deployer) runTarget(ctx context.Context, model *ir.Model, projectID string, target store.Target, credential secrets.Secret, batch int, dryRun bool) []Step {
	generated, err := validate.Generate(model, target.Engine)
	steps := []Step{{
		TargetID:   target.ID,
		TargetName: target.Name,
		Engine:     target.Engine,
		Stage:      "generate",
		Status:     stepStatus(err, dryRun),
		Message:    errString(err),
		Batch:      batch,
	}}
	if err != nil {
		return steps
	}
	remoteTmp := fmt.Sprintf("/tmp/mizan-%s-%s.cfg", projectID, target.ID)
	commands := []struct {
		stage string
		cmd   string
		input string
	}{
		{"upload", uploadCommand(target, remoteTmp), generated.Config},
		{"remote_validate", remoteValidateCommand(target, remoteTmp), ""},
		{"install", installCommand(target, remoteTmp), ""},
		{"reload", reloadCommand(target), ""},
	}
	for _, item := range commands {
		step := Step{TargetID: target.ID, TargetName: target.Name, Engine: target.Engine, Stage: item.stage, Command: item.cmd, Batch: batch}
		if dryRun {
			step.Status = "skipped"
			step.Message = "dry run"
		} else if output, runErr := d.Runner(ctx, target, credential, item.cmd, item.input); runErr != nil {
			step.Credential = credentialSource(credential)
			step.Status = "failed"
			step.Message = strings.TrimSpace(output + "\n" + runErr.Error())
			steps = append(steps, step)
			if target.RollbackCommand != "" && (item.stage == "install" || item.stage == "reload") {
				steps = append(steps, d.rollbackStep(ctx, target, credential, batch, dryRun))
			}
			steps = append(steps, d.cleanupStep(ctx, target, credential, batch, dryRun, remoteTmp))
			return steps
		} else {
			step.Credential = credentialSource(credential)
			step.Status = "success"
			step.Message = strings.TrimSpace(output)
		}
		steps = append(steps, step)
	}
	if target.PostReloadProbe != "" {
		step := Step{TargetID: target.ID, TargetName: target.Name, Engine: target.Engine, Stage: "probe", Command: target.PostReloadProbe, Batch: batch}
		if dryRun {
			step.Status = "skipped"
			step.Message = "dry run"
		} else if err := d.Prober(ctx, target.PostReloadProbe); err != nil {
			step.Status = "failed"
			step.Message = err.Error()
			steps = append(steps, step)
			if target.RollbackCommand != "" {
				steps = append(steps, d.rollbackStep(ctx, target, credential, batch, dryRun))
			}
			steps = append(steps, d.cleanupStep(ctx, target, credential, batch, dryRun, remoteTmp))
			return steps
		} else {
			step.Status = "success"
		}
		steps = append(steps, step)
	}
	if dryRun && target.RollbackCommand != "" {
		steps = append(steps, d.rollbackStep(ctx, target, credential, batch, dryRun))
	}
	steps = append(steps, d.cleanupStep(ctx, target, credential, batch, dryRun, remoteTmp))
	return steps
}

func (d Deployer) cleanupStep(ctx context.Context, target store.Target, credential secrets.Secret, batch int, dryRun bool, remoteTmp string) Step {
	cleanup := Step{TargetID: target.ID, TargetName: target.Name, Engine: target.Engine, Stage: "cleanup", Command: cleanupCommand(remoteTmp), Batch: batch}
	if dryRun {
		cleanup.Status = "skipped"
		cleanup.Message = "dry run"
	} else if output, runErr := d.Runner(ctx, target, credential, cleanup.Command, ""); runErr != nil {
		cleanup.Credential = credentialSource(credential)
		cleanup.Status = "failed"
		cleanup.Message = strings.TrimSpace(output + "\n" + runErr.Error())
	} else {
		cleanup.Credential = credentialSource(credential)
		cleanup.Status = "success"
		cleanup.Message = strings.TrimSpace(output)
	}
	return cleanup
}

func (d Deployer) rollbackStep(ctx context.Context, target store.Target, credential secrets.Secret, batch int, dryRun bool) Step {
	step := Step{
		TargetID:   target.ID,
		TargetName: target.Name,
		Engine:     target.Engine,
		Stage:      "rollback",
		Command:    rollbackCommand(target),
		Batch:      batch,
	}
	if target.RollbackCommand == "" {
		step.Status = "skipped"
		step.Message = "no rollback command configured"
		return step
	}
	if dryRun {
		step.Status = "skipped"
		step.Message = "runs after failed install, reload, or probe"
		return step
	}
	output, err := d.Runner(ctx, target, credential, step.Command, "")
	step.Credential = credentialSource(credential)
	step.Message = strings.TrimSpace(output)
	if err != nil {
		step.Status = "failed"
		step.Message = strings.TrimSpace(output + "\n" + err.Error())
		return step
	}
	step.Status = "success"
	return step
}

func selectTargets(file store.TargetsFile, req Request) ([]store.Target, int, bool, int, error) {
	if req.TargetID != "" {
		for _, target := range file.Targets {
			if target.ID == req.TargetID {
				return []store.Target{target}, 1, true, 0, nil
			}
		}
		return nil, 0, false, 0, errors.New("target not found")
	}
	for _, cluster := range file.Clusters {
		if cluster.ID == req.ClusterID {
			byID := map[string]store.Target{}
			for _, target := range file.Targets {
				byID[target.ID] = target
			}
			selected := make([]store.Target, 0, len(cluster.TargetIDs))
			for _, id := range cluster.TargetIDs {
				target, ok := byID[id]
				if !ok {
					return nil, 0, false, 0, errors.New("cluster references a missing target")
				}
				selected = append(selected, target)
			}
			if len(selected) == 0 {
				return nil, 0, false, 0, errors.New("cluster has no targets")
			}
			if cluster.Parallelism <= 0 {
				cluster.Parallelism = 1
			}
			if cluster.RequiredApprovals < 0 {
				cluster.RequiredApprovals = 0
			}
			return selected, cluster.Parallelism, cluster.GateOnFailure, cluster.RequiredApprovals, nil
		}
	}
	return nil, 0, false, 0, errors.New("cluster not found")
}

func uploadCommand(target store.Target, remoteTmp string) string {
	return "cat > " + shellQuote(remoteTmp)
}

func remoteValidateCommand(target store.Target, remoteTmp string) string {
	if target.Engine == ir.EngineNginx {
		return "nginx -t -c " + shellQuote(remoteTmp)
	}
	return "haproxy -c -f " + shellQuote(remoteTmp)
}

func installCommand(target store.Target, remoteTmp string) string {
	cmd := fmt.Sprintf("install -m 0644 %s %s", shellQuote(remoteTmp), shellQuote(target.ConfigPath))
	return sudoCommand(target, cmd)
}

func reloadCommand(target store.Target) string {
	return sudoCommand(target, target.ReloadCommand)
}

func rollbackCommand(target store.Target) string {
	return sudoCommand(target, target.RollbackCommand)
}

func cleanupCommand(remoteTmp string) string {
	return "rm -f " + shellQuote(remoteTmp)
}

func sudoCommand(target store.Target, command string) string {
	if !target.Sudo {
		return command
	}
	return "sudo sh -lc " + shellQuote(command)
}

func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "'\"'\"'") + "'"
}

func sshRunner(ctx context.Context, target store.Target, credential secrets.Secret, command string, input string) (string, error) {
	if credential.Username != "" {
		target.User = credential.Username
	}
	keyPath, cleanup, err := privateKeyFile(credential)
	if err != nil {
		return "", err
	}
	defer cleanup()
	args := []string{}
	if keyPath != "" {
		args = append(args, "-i", keyPath, "-o", "IdentitiesOnly=yes")
	}
	args = append(args, "-p", fmt.Sprint(target.Port), target.User+"@"+target.Host, command)
	cmd := exec.CommandContext(ctx, "ssh", args...)
	if input != "" {
		cmd.Stdin = strings.NewReader(input)
	}
	var output bytes.Buffer
	cmd.Stdout = &output
	cmd.Stderr = &output
	err = cmd.Run()
	return output.String(), err
}

func privateKeyFile(credential secrets.Secret) (string, func(), error) {
	if credential.PrivateKey == "" {
		return "", func() {}, nil
	}
	path, err := createTempKeyFile()
	if err != nil {
		return "", nil, err
	}
	if err := writeKeyFile(path, []byte(credential.PrivateKey), 0o600); err != nil {
		_ = removeKeyFile(path)
		return "", nil, err
	}
	if err := chmodKeyFile(path, 0o600); err != nil {
		_ = removeKeyFile(path)
		return "", nil, err
	}
	return path, func() { _ = removeKeyFile(path) }, nil
}

func httpProbe(ctx context.Context, url string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode < 200 || res.StatusCode >= 400 {
		return fmt.Errorf("probe returned HTTP %d", res.StatusCode)
	}
	return nil
}

func ProbeTarget(ctx context.Context, target store.Target, prober Prober, now func() time.Time) (ProbeResult, error) {
	url := target.PostReloadProbe
	if url == "" {
		url = target.MonitorEndpoint
	}
	if url == "" {
		return ProbeResult{}, errors.New("target has no probe or monitor endpoint")
	}
	if prober == nil {
		prober = httpProbe
	}
	if now == nil {
		now = func() time.Time { return time.Now().UTC() }
	}
	result := ProbeResult{
		TargetID:   target.ID,
		TargetName: target.Name,
		URL:        url,
		Status:     "success",
		CheckedAt:  now().UTC().Format(time.RFC3339),
	}
	if err := prober(ctx, url); err != nil {
		result.Status = "failed"
		result.Message = err.Error()
	}
	return result, nil
}

func stepStatus(err error, dryRun bool) string {
	if err != nil {
		return "failed"
	}
	if dryRun {
		return "success"
	}
	return "success"
}

func errString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

func hasFailed(steps []Step) bool {
	for _, step := range steps {
		if step.Status == "failed" {
			return true
		}
	}
	return false
}

func credentialSource(credential secrets.Secret) string {
	if credential.Username != "" || credential.PrivateKey != "" || credential.Password != "" || credential.Passphrase != "" || credential.Token != "" {
		return "vault"
	}
	return "local_ssh"
}

func CredentialSources(steps []Step) []string {
	seen := map[string]bool{}
	sources := []string{}
	for _, step := range steps {
		if step.Credential == "" || seen[step.Credential] {
			continue
		}
		seen[step.Credential] = true
		sources = append(sources, step.Credential)
	}
	return sources
}

func RollbackSummary(steps []Step) RollbackStats {
	var stats RollbackStats
	for _, step := range steps {
		if step.Stage != "rollback" {
			continue
		}
		stats.Planned++
		if step.Status == "skipped" {
			continue
		}
		stats.Attempted++
		if step.Status == "failed" {
			stats.Failed++
		} else if step.Status == "success" {
			stats.Succeeded++
		}
	}
	return stats
}

func CleanupSummary(steps []Step) CleanupStats {
	var stats CleanupStats
	for _, step := range steps {
		if step.Stage != "cleanup" {
			continue
		}
		stats.Planned++
		if step.Status == "skipped" {
			continue
		}
		stats.Attempted++
		if step.Status == "failed" {
			stats.Failed++
		} else if step.Status == "success" {
			stats.Succeeded++
		}
	}
	return stats
}

func credentialFailureStep(target store.Target, batch int, err error) Step {
	return Step{
		TargetID:   target.ID,
		TargetName: target.Name,
		Engine:     target.Engine,
		Stage:      "credentials",
		Status:     "failed",
		Message:    err.Error(),
		Batch:      batch,
	}
}
