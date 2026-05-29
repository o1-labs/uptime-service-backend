package delegation_backend

import (
	"encoding/json"
	"net/http/httptest"
	"testing"
	"time"

	logging "github.com/ipfs/go-log/v2"
)

func TestOnlineStorageServeHTTP(t *testing.T) {
	log := logging.Logger("delegation backend online test")
	now := time.Date(2026, 5, 28, 12, 0, 0, 0, time.UTC)
	storage := NewOnlineStorage(log, func() time.Time { return now })
	storage.Record(OnlineRecord{
		RemoteAddr:         "10.0.0.2",
		Submitter:          "B62valid",
		GraphqlControlPort: 10001,
	}, now.Add(-5*time.Minute))
	storage.Record(OnlineRecord{
		RemoteAddr:         "10.0.0.3",
		Submitter:          "B62expired",
		GraphqlControlPort: 10002,
	}, now.Add(-ONLINE_KEEP_INTERVAL-time.Second))

	req := httptest.NewRequest("GET", "/v1/online", nil)
	rr := httptest.NewRecorder()
	storage.ServeHTTP(rr, req)

	if rr.Code != 200 {
		t.Fatalf("unexpected status code: got %d want 200", rr.Code)
	}

	var records []OnlineRecord
	if err := json.Unmarshal(rr.Body.Bytes(), &records); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("unexpected number of online records: got %d want 1", len(records))
	}
	if records[0].Submitter != "B62valid" || records[0].RemoteAddr != "10.0.0.2" || records[0].GraphqlControlPort != 10001 {
		t.Fatalf("unexpected response payload: %+v", records[0])
	}
}

func TestNormalizeRemoteAddr(t *testing.T) {
	testCases := []struct {
		name  string
		input string
		want  string
	}{
		{name: "host port", input: "192.0.2.1:1234", want: "192.0.2.1"},
		{name: "xff first value", input: "10.0.0.1:1234, 10.0.0.2", want: "10.0.0.1"},
		{name: "plain ip", input: "5.189.147.209", want: "5.189.147.209"},
		{name: "ipv6 host port", input: "[2001:db8::1]:8080", want: "2001:db8::1"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := normalizeRemoteAddr(tc.input)
			if got != tc.want {
				t.Fatalf("unexpected normalized address: got %q want %q", got, tc.want)
			}
		})
	}
}

func TestRequestRemoteAddr(t *testing.T) {
	testCases := []struct {
		name       string
		remoteAddr string
		xff        string
		want       string
	}{
		{name: "no xff falls back to remote addr", remoteAddr: "192.0.2.1:1234", want: "192.0.2.1:1234"},
		{name: "uses public xff hop", remoteAddr: "127.0.0.1:9999", xff: "127.0.0.1, 203.0.113.9", want: "203.0.113.9"},
		{name: "uses first public xff after private hops", remoteAddr: "127.0.0.1:9999", xff: "10.0.0.4, 172.16.0.8, 198.51.100.7", want: "198.51.100.7"},
		{name: "falls back to first valid xff when no public hop", remoteAddr: "127.0.0.1:9999", xff: "127.0.0.1, 10.0.0.4", want: "127.0.0.1"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/v1/submit", nil)
			req.RemoteAddr = tc.remoteAddr
			if tc.xff != "" {
				req.Header.Set("X-Forwarded-For", tc.xff)
			}
			got := requestRemoteAddr(req)
			if got != tc.want {
				t.Fatalf("unexpected request remote addr: got %q want %q", got, tc.want)
			}
		})
	}
}
