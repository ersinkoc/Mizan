package doctor

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/mizanproxy/mizan/internal/ir"
	"github.com/mizanproxy/mizan/internal/store"
)

type Status string

const (
	StatusPass Status = "pass"
	StatusWarn Status = "warn"
	StatusFail Status = "fail"
)

type Check struct {
	Name    string `json:"name"`
	Status  Status `json:"status"`
	Message string `json:"message"`
}

type Report struct {
	Status       Status  `json:"status"`
	Root         string  `json:"root"`
	ProjectCount int     `json:"project_count"`
	TargetCount  int     `json:"target_count"`
	ClusterCount int     `json:"cluster_count"`
	Checks       []Check `json:"checks"`
}

type LookPath func(string) (string, error)

func Run(ctx context.Context, st *store.Store, lookPath LookPath) Report {
	if lookPath == nil {
		lookPath = exec.LookPath
	}
	report := Report{Status: StatusPass, Root: st.Root()}
	add := func(name string, status Status, message string) {
		report.Checks = append(report.Checks, Check{Name: name, Status: status, Message: message})
		report.Status = worst(report.Status, status)
	}

	if err := st.Bootstrap(ctx); err != nil {
		add("data_root", StatusFail, err.Error())
		return report
	}
	add("data_root", StatusPass, fmt.Sprintf("%s is accessible", st.Root()))

	projects, err := st.ListProjects(ctx)
	if err != nil {
		add("projects", StatusFail, err.Error())
		return report
	}
	report.ProjectCount = len(projects)
	add("projects", StatusPass, fmt.Sprintf("%d project(s) readable", len(projects)))

	engines := map[ir.Engine]bool{}
	var integrityErrors []string
	for _, project := range projects {
		for _, engine := range project.Engines {
			engines[engine] = true
		}
		if _, _, err := st.GetIR(ctx, project.ID); err != nil {
			integrityErrors = append(integrityErrors, project.ID+": "+err.Error())
		}
		targets, err := st.ListTargets(ctx, project.ID)
		if err != nil {
			integrityErrors = append(integrityErrors, project.ID+" targets: "+err.Error())
			continue
		}
		report.TargetCount += len(targets.Targets)
		report.ClusterCount += len(targets.Clusters)
	}
	if len(integrityErrors) > 0 {
		add("project_integrity", StatusFail, strings.Join(integrityErrors, "; "))
	} else {
		add("project_integrity", StatusPass, "project config and target files are readable")
	}
	add("targets", StatusPass, fmt.Sprintf("%d target(s), %d cluster(s)", report.TargetCount, report.ClusterCount))

	checkSecrets(st.Root(), add)
	checkTool("ssh", "remote deployment execution", true, lookPath, add)
	checkNativeTool(ir.EngineHAProxy, "haproxy", engines[ir.EngineHAProxy], lookPath, add)
	checkNativeTool(ir.EngineNginx, "nginx", engines[ir.EngineNginx], lookPath, add)
	return report
}

func checkSecrets(root string, add func(string, Status, string)) {
	secretRoot := filepath.Join(root, "secrets")
	info, err := os.Stat(secretRoot)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			add("secrets", StatusPass, "no encrypted secrets stored")
			return
		}
		add("secrets", StatusFail, err.Error())
		return
	}
	if !info.IsDir() {
		add("secrets", StatusFail, fmt.Sprintf("%s is not a directory", secretRoot))
		return
	}
	entries, err := os.ReadDir(secretRoot)
	if err != nil {
		add("secrets", StatusFail, err.Error())
		return
	}
	count := 0
	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".json" {
			count++
		}
	}
	add("secrets", StatusPass, fmt.Sprintf("%d encrypted secret envelope(s)", count))
}

func checkTool(name, purpose string, important bool, lookPath LookPath, add func(string, Status, string)) {
	path, err := lookPath(name)
	if err != nil {
		status := StatusWarn
		if important {
			status = StatusWarn
		}
		add("tool_"+name, status, fmt.Sprintf("%s not found on PATH; %s will be limited", name, purpose))
		return
	}
	add("tool_"+name, StatusPass, path)
}

func checkNativeTool(engine ir.Engine, tool string, used bool, lookPath LookPath, add func(string, Status, string)) {
	path, err := lookPath(tool)
	if err != nil {
		if used {
			add("native_"+tool, StatusWarn, fmt.Sprintf("%s project(s) exist but %s is not on PATH; native validation will be skipped", engine, tool))
			return
		}
		add("native_"+tool, StatusPass, fmt.Sprintf("%s not on PATH; no %s projects currently require it", tool, engine))
		return
	}
	add("native_"+tool, StatusPass, path)
}

func worst(current, next Status) Status {
	if current == StatusFail || next == StatusFail {
		return StatusFail
	}
	if current == StatusWarn || next == StatusWarn {
		return StatusWarn
	}
	return StatusPass
}
