package delegation_backend

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"sync"
	"time"

	logging "github.com/ipfs/go-log/v2"
)

type OnlineRecord struct {
	RemoteAddr         string `json:"remote_addr"`
	Submitter          string `json:"submitter"`
	GraphqlControlPort int    `json:"graphql_control_port,omitempty"`
}

type onlineEntry struct {
	record      OnlineRecord
	submittedAt time.Time
}

type OnlineStorage struct {
	mu   sync.Mutex
	data map[string]onlineEntry
	now  nowFunc
	log  *logging.ZapEventLogger
}

func NewOnlineStorage(log *logging.ZapEventLogger, now nowFunc) *OnlineStorage {
	return &OnlineStorage{
		data: make(map[string]onlineEntry),
		now:  now,
		log:  log,
	}
}

func (storage *OnlineStorage) Record(record OnlineRecord, submittedAt time.Time) {
	storage.mu.Lock()
	defer storage.mu.Unlock()
	storage.data[record.Submitter] = onlineEntry{
		record:      record,
		submittedAt: submittedAt,
	}
}

func (storage *OnlineStorage) ServeHTTP(w http.ResponseWriter, _ *http.Request) {
	result := storage.recentRecords()
	bs, err := json.Marshal(result)
	if err != nil {
		storage.log.Errorf("Error while marshaling online response: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = io.Copy(w, bytes.NewReader([]byte(`{"error":"Unexpected server error"}`)))
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_, err = io.Copy(w, bytes.NewReader(bs))
	if err != nil {
		storage.log.Debugf("Error while responding with /v1/online payload: %v", err)
	}
}

func (storage *OnlineStorage) recentRecords() []OnlineRecord {
	storage.mu.Lock()
	defer storage.mu.Unlock()

	earliest := storage.now().Add(-ONLINE_KEEP_INTERVAL)
	result := make([]OnlineRecord, 0, len(storage.data))
	for submitter, entry := range storage.data {
		if entry.submittedAt.After(earliest) {
			result = append(result, entry.record)
			continue
		}
		delete(storage.data, submitter)
	}
	return result
}
