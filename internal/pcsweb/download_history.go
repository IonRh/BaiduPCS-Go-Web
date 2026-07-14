package pcsweb

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	"github.com/qjfoidnh/BaiduPCS-Go/internal/pcsconfig"
)

const downloadHistoryFileName = "pcsweb_download_history.json"

type downloadHistoryEntry struct {
	ID         string `json:"id"`
	SessionID  string `json:"-"`
	Path       string `json:"path"`
	SavePath   string `json:"save_path"`
	Status     string `json:"status"`
	StartedAt  int64  `json:"started_at"`
	FinishedAt int64  `json:"finished_at,omitempty"`
	Error      string `json:"error,omitempty"`
}

var downloadHistoryMu sync.Mutex

func downloadHistoryPath() string {
	return filepath.Join(pcsconfig.GetConfigDir(), downloadHistoryFileName)
}

func readDownloadHistory() ([]downloadHistoryEntry, error) {
	data, err := os.ReadFile(downloadHistoryPath())
	if os.IsNotExist(err) {
		return []downloadHistoryEntry{}, nil
	}
	if err != nil {
		return nil, err
	}
	if len(data) == 0 {
		return []downloadHistoryEntry{}, nil
	}
	var history []downloadHistoryEntry
	if err := json.Unmarshal(data, &history); err != nil {
		return nil, err
	}
	return history, nil
}

func writeDownloadHistory(history []downloadHistoryEntry) error {
	if err := os.MkdirAll(filepath.Dir(downloadHistoryPath()), 0700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(history, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(downloadHistoryPath(), data, 0600)
}

func beginDownloadHistory(sessionID, remotePath, savePath string) (string, error) {
	downloadHistoryMu.Lock()
	defer downloadHistoryMu.Unlock()
	history, err := readDownloadHistory()
	if err != nil {
		return "", err
	}
	id := fmt.Sprintf("%d", time.Now().UnixNano())
	history = append([]downloadHistoryEntry{{
		ID:        id,
		SessionID: sessionID,
		Path:      remotePath,
		SavePath:  savePath,
		Status:    "下载中",
		StartedAt: time.Now().Unix(),
	}}, history...)
	if len(history) > 100 {
		history = history[:100]
	}
	return id, writeDownloadHistory(history)
}

func finishDownloadHistory(id, status, downloadErr string) {
	downloadHistoryMu.Lock()
	defer downloadHistoryMu.Unlock()
	history, err := readDownloadHistory()
	if err != nil {
		return
	}
	for index := range history {
		if history[index].ID != id {
			continue
		}
		history[index].Status = status
		history[index].Error = downloadErr
		history[index].FinishedAt = time.Now().Unix()
		break
	}
	_ = writeDownloadHistory(history)
}

func listDownloadHistory() ([]downloadHistoryEntry, error) {
	downloadHistoryMu.Lock()
	defer downloadHistoryMu.Unlock()
	history, err := readDownloadHistory()
	if err != nil {
		return nil, err
	}
	return deduplicateBrowserDownloadHistory(history), nil
}

func deduplicateBrowserDownloadHistory(history []downloadHistoryEntry) []downloadHistoryEntry {
	result := make([]downloadHistoryEntry, 0, len(history))
	seen := make(map[string]struct{})
	for _, item := range history {
		key := item.ID
		if item.SavePath == "浏览器下载" {
			key = item.Path + "\x00" + item.SavePath + "\x00" + strconv.FormatInt(item.StartedAt, 10)
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, item)
	}
	return result
}
