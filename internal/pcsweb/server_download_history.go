package pcsweb

import (
	"fmt"
	"sync"
	"time"
)

const serverDownloadHistoryWindow = 2 * time.Minute

type serverDownloadHistoryRecord struct {
	ID       string
	LastSeen time.Time
}

var serverDownloadHistoryMu sync.Mutex
var serverDownloadHistories = make(map[string]serverDownloadHistoryRecord)

// Download managers may open several range requests for one browser download.
// Reuse one history entry while those requests are active.
func beginServerDownloadHistory(sessionID, localPath string, size int64, modifiedAt time.Time) (string, error) {
	now := time.Now()
	key := fmt.Sprintf("%s\x00%s\x00%d\x00%d", sessionID, localPath, size, modifiedAt.UnixNano())

	serverDownloadHistoryMu.Lock()
	defer serverDownloadHistoryMu.Unlock()
	for existingKey, record := range serverDownloadHistories {
		if now.Sub(record.LastSeen) > serverDownloadHistoryWindow {
			delete(serverDownloadHistories, existingKey)
		}
	}
	if record, ok := serverDownloadHistories[key]; ok {
		record.LastSeen = now
		serverDownloadHistories[key] = record
		return record.ID, nil
	}

	id, err := beginDownloadHistory(sessionID, localPath, "浏览器下载")
	if err != nil {
		return "", err
	}
	serverDownloadHistories[key] = serverDownloadHistoryRecord{ID: id, LastSeen: now}
	return id, nil
}
