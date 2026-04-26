package observe

import (
	"fmt"
	"io"
	"sort"
	"strings"
	"sync"
)

type httpRequestKey struct {
	Method string
	Route  string
	Status int
}

type HTTPRequestMetric struct {
	Method string
	Route  string
	Status int
	Count  uint64
}

var (
	httpMu       sync.Mutex
	httpRequests = map[httpRequestKey]uint64{}
)

func RecordHTTPRequest(method, route string, status int) {
	if method == "" {
		method = "UNKNOWN"
	}
	if route == "" {
		route = "unmatched"
	}
	if status == 0 {
		status = 200
	}
	httpMu.Lock()
	httpRequests[httpRequestKey{Method: method, Route: route, Status: status}]++
	httpMu.Unlock()
}

func HTTPRequestSnapshot() []HTTPRequestMetric {
	httpMu.Lock()
	defer httpMu.Unlock()
	out := make([]HTTPRequestMetric, 0, len(httpRequests))
	for key, count := range httpRequests {
		out = append(out, HTTPRequestMetric{
			Method: key.Method,
			Route:  key.Route,
			Status: key.Status,
			Count:  count,
		})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Route != out[j].Route {
			return out[i].Route < out[j].Route
		}
		if out[i].Method != out[j].Method {
			return out[i].Method < out[j].Method
		}
		return out[i].Status < out[j].Status
	})
	return out
}

func WriteHTTPMetrics(w io.Writer) {
	_, _ = fmt.Fprintln(w, "# HELP mizan_http_requests_total HTTP requests served by method, route pattern, and status code.")
	_, _ = fmt.Fprintln(w, "# TYPE mizan_http_requests_total counter")
	for _, metric := range HTTPRequestSnapshot() {
		_, _ = fmt.Fprintf(w, "mizan_http_requests_total{method=\"%s\",route=\"%s\",status=\"%d\"} %d\n",
			prometheusLabelValue(metric.Method),
			prometheusLabelValue(metric.Route),
			metric.Status,
			metric.Count,
		)
	}
}

func prometheusLabelValue(value string) string {
	return strings.NewReplacer(`\`, `\\`, "\n", `\n`, `"`, `\"`).Replace(value)
}

func resetHTTPRequestsForTest() {
	httpMu.Lock()
	defer httpMu.Unlock()
	httpRequests = map[httpRequestKey]uint64{}
}
