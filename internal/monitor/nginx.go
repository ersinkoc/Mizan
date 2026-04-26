package monitor

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/mizanproxy/mizan/internal/store"
)

type NginxStubStatus struct {
	Active   int `json:"active"`
	Accepts  int `json:"accepts"`
	Handled  int `json:"handled"`
	Requests int `json:"requests"`
	Reading  int `json:"reading"`
	Writing  int `json:"writing"`
	Waiting  int `json:"waiting"`
}

func CollectNginx(ctx context.Context, target store.Target) (TargetSnapshot, error) {
	if target.MonitorEndpoint == "" {
		return baseTargetSnapshot(target, "unknown", "runtime collector is not configured"), nil
	}
	data, err := fetchURL(ctx, target.MonitorEndpoint)
	if err != nil {
		return baseTargetSnapshot(target, "failed", err.Error()), nil
	}
	stats, err := ParseNginxStubStatus(string(data))
	if err != nil {
		return baseTargetSnapshot(target, "failed", err.Error()), nil
	}
	status, message := summarizeNginx(stats)
	return baseTargetSnapshot(target, status, message), nil
}

func ParseNginxStubStatus(data string) (NginxStubStatus, error) {
	lines := nonEmptyLines(data)
	if len(lines) < 4 {
		return NginxStubStatus{}, errors.New("nginx stub_status payload is incomplete")
	}
	active, err := parsePrefixedInt(lines[0], "Active connections:")
	if err != nil {
		return NginxStubStatus{}, err
	}
	accepts, handled, requests, err := parseNginxCounters(lines[2])
	if err != nil {
		return NginxStubStatus{}, err
	}
	reading, err := parseNamedMetric(lines[3], "Reading:")
	if err != nil {
		return NginxStubStatus{}, err
	}
	writing, err := parseNamedMetric(lines[3], "Writing:")
	if err != nil {
		return NginxStubStatus{}, err
	}
	waiting, err := parseNamedMetric(lines[3], "Waiting:")
	if err != nil {
		return NginxStubStatus{}, err
	}
	return NginxStubStatus{
		Active:   active,
		Accepts:  accepts,
		Handled:  handled,
		Requests: requests,
		Reading:  reading,
		Writing:  writing,
		Waiting:  waiting,
	}, nil
}

func summarizeNginx(stats NginxStubStatus) (string, string) {
	if stats.Handled < stats.Accepts {
		return "warning", fmt.Sprintf("%d accepted Nginx connections were not handled", stats.Accepts-stats.Handled)
	}
	return "healthy", fmt.Sprintf("nginx active=%d reading=%d writing=%d waiting=%d requests=%d", stats.Active, stats.Reading, stats.Writing, stats.Waiting, stats.Requests)
}

func nonEmptyLines(data string) []string {
	var lines []string
	for _, line := range strings.Split(data, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			lines = append(lines, line)
		}
	}
	return lines
}

func parsePrefixedInt(line string, prefix string) (int, error) {
	if !strings.HasPrefix(line, prefix) {
		return 0, fmt.Errorf("nginx stub_status expected %q", prefix)
	}
	value, err := strconv.Atoi(strings.TrimSpace(strings.TrimPrefix(line, prefix)))
	if err != nil {
		return 0, err
	}
	return value, nil
}

func parseNginxCounters(line string) (int, int, int, error) {
	fields := strings.Fields(line)
	if len(fields) != 3 {
		return 0, 0, 0, errors.New("nginx stub_status counter line is invalid")
	}
	accepts, err := strconv.Atoi(fields[0])
	if err != nil {
		return 0, 0, 0, err
	}
	handled, err := strconv.Atoi(fields[1])
	if err != nil {
		return 0, 0, 0, err
	}
	requests, err := strconv.Atoi(fields[2])
	if err != nil {
		return 0, 0, 0, err
	}
	return accepts, handled, requests, nil
}

func parseNamedMetric(line string, label string) (int, error) {
	fields := strings.Fields(line)
	for i := 0; i+1 < len(fields); i++ {
		if fields[i] == label {
			value, err := strconv.Atoi(fields[i+1])
			if err != nil {
				return 0, err
			}
			return value, nil
		}
	}
	return 0, fmt.Errorf("nginx stub_status metric %q is missing", label)
}
