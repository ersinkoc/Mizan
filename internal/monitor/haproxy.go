package monitor

import (
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/mizanproxy/mizan/internal/store"
)

type HAProxyStats struct {
	Servers []HAProxyServerStat `json:"servers"`
}

type HAProxyServerStat struct {
	ProxyName  string `json:"proxy_name"`
	ServerName string `json:"server_name"`
	Status     string `json:"status"`
	Current    string `json:"current_sessions,omitempty"`
}

var fetchURL = func(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode < 200 || res.StatusCode >= 400 {
		return nil, fmt.Errorf("monitor endpoint returned HTTP %d", res.StatusCode)
	}
	return io.ReadAll(res.Body)
}

func CollectHAProxy(ctx context.Context, target store.Target) (TargetSnapshot, error) {
	if target.MonitorEndpoint == "" {
		return baseTargetSnapshot(target, "unknown", "runtime collector is not configured"), nil
	}
	data, err := fetchURL(ctx, target.MonitorEndpoint)
	if err != nil {
		return baseTargetSnapshot(target, "failed", err.Error()), nil
	}
	stats, err := ParseHAProxyStats(string(data))
	if err != nil {
		return baseTargetSnapshot(target, "failed", err.Error()), nil
	}
	status, message := summarizeHAProxy(stats)
	return baseTargetSnapshot(target, status, message), nil
}

func ParseHAProxyStats(data string) (HAProxyStats, error) {
	reader := csv.NewReader(strings.NewReader(data))
	reader.FieldsPerRecord = -1
	var headers []string
	var stats HAProxyStats
	for {
		record, err := reader.Read()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return HAProxyStats{}, err
		}
		if strings.HasPrefix(record[0], "# ") {
			record[0] = strings.TrimPrefix(record[0], "# ")
			headers = record
			continue
		}
		if headers == nil {
			return HAProxyStats{}, errors.New("haproxy stats header is missing")
		}
		row := map[string]string{}
		for i, header := range headers {
			if i < len(record) {
				row[header] = record[i]
			}
		}
		name := row["svname"]
		if name == "" || name == "FRONTEND" || name == "BACKEND" {
			continue
		}
		stats.Servers = append(stats.Servers, HAProxyServerStat{
			ProxyName:  row["pxname"],
			ServerName: name,
			Status:     strings.ToUpper(row["status"]),
			Current:    row["scur"],
		})
	}
	if headers == nil {
		return HAProxyStats{}, errors.New("haproxy stats header is missing")
	}
	return stats, nil
}

func summarizeHAProxy(stats HAProxyStats) (string, string) {
	if len(stats.Servers) == 0 {
		return "unknown", "haproxy stats did not include server rows"
	}
	down := 0
	warn := 0
	up := 0
	for _, server := range stats.Servers {
		switch server.Status {
		case "UP":
			up++
		case "MAINT", "DRAIN", "NOLB":
			warn++
		default:
			down++
		}
	}
	switch {
	case down > 0:
		return "failed", fmt.Sprintf("%d/%d HAProxy servers are down", down, len(stats.Servers))
	case warn > 0:
		return "warning", fmt.Sprintf("%d/%d HAProxy servers need attention", warn, len(stats.Servers))
	default:
		return "healthy", fmt.Sprintf("%d HAProxy servers are up", up)
	}
}

func baseTargetSnapshot(target store.Target, status string, message string) TargetSnapshot {
	return TargetSnapshot{
		TargetID: target.ID,
		Name:     target.Name,
		Host:     target.Host,
		Engine:   target.Engine,
		Status:   status,
		Message:  message,
	}
}
