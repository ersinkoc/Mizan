package observe

import (
	"bytes"
	"strings"
	"testing"
)

func TestHTTPRequestMetrics(t *testing.T) {
	resetHTTPRequestsForTest()
	RecordHTTPRequest("GET", "/version", 200)
	RecordHTTPRequest("GET", "/version", 200)
	RecordHTTPRequest("POST", "/version", 200)
	RecordHTTPRequest("POST", "/version", 500)
	RecordHTTPRequest("POST", `/api/"quoted"\path`, 201)
	RecordHTTPRequest("", "", 0)

	snapshot := HTTPRequestSnapshot()
	if len(snapshot) != 5 {
		t.Fatalf("snapshot len=%d", len(snapshot))
	}
	if snapshot[0].Route != "/api/\"quoted\"\\path" || snapshot[0].Method != "POST" || snapshot[0].Status != 201 || snapshot[0].Count != 1 {
		t.Fatalf("first snapshot entry=%+v", snapshot[0])
	}
	if snapshot[1].Route != "/version" || snapshot[1].Method != "GET" || snapshot[1].Status != 200 || snapshot[1].Count != 2 {
		t.Fatalf("second snapshot entry=%+v", snapshot[1])
	}
	if snapshot[2].Route != "/version" || snapshot[2].Method != "POST" || snapshot[2].Status != 200 || snapshot[2].Count != 1 {
		t.Fatalf("third snapshot entry=%+v", snapshot[2])
	}
	if snapshot[3].Route != "/version" || snapshot[3].Method != "POST" || snapshot[3].Status != 500 || snapshot[3].Count != 1 {
		t.Fatalf("fourth snapshot entry=%+v", snapshot[3])
	}
	if snapshot[4].Route != "unmatched" || snapshot[4].Method != "UNKNOWN" || snapshot[4].Status != 200 || snapshot[4].Count != 1 {
		t.Fatalf("fifth snapshot entry=%+v", snapshot[4])
	}

	var buf bytes.Buffer
	WriteHTTPMetrics(&buf)
	text := buf.String()
	for _, want := range []string{
		"# TYPE mizan_http_requests_total counter",
		`mizan_http_requests_total{method="GET",route="/version",status="200"} 2`,
		`mizan_http_requests_total{method="POST",route="/version",status="200"} 1`,
		`mizan_http_requests_total{method="POST",route="/version",status="500"} 1`,
		`mizan_http_requests_total{method="POST",route="/api/\"quoted\"\\path",status="201"} 1`,
		`mizan_http_requests_total{method="UNKNOWN",route="unmatched",status="200"} 1`,
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("metrics missing %q in:\n%s", want, text)
		}
	}
}
