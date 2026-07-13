package pcsweb

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/qjfoidnh/BaiduPCS-Go/internal/pcsconfig"
)

const uploadHistoryFileName = "pcsweb_upload_history.json"

type uploadHistoryEntry struct {
	ID         string   `json:"id"`
	SessionID  string   `json:"-"`
	Files      []string `json:"files"`
	TargetPath string   `json:"target_path"`
	Status     string   `json:"status"`
	StartedAt  int64    `json:"started_at"`
	FinishedAt int64    `json:"finished_at,omitempty"`
	Error      string   `json:"error,omitempty"`
}

var uploadHistoryMu sync.Mutex

func uploadHistoryPath() string {
	return filepath.Join(pcsconfig.GetConfigDir(), uploadHistoryFileName)
}

func readUploadHistory() ([]uploadHistoryEntry, error) {
	data, err := os.ReadFile(uploadHistoryPath())
	if os.IsNotExist(err) {
		return []uploadHistoryEntry{}, nil
	}
	if err != nil {
		return nil, err
	}
	if len(data) == 0 {
		return []uploadHistoryEntry{}, nil
	}
	var history []uploadHistoryEntry
	if err := json.Unmarshal(data, &history); err != nil {
		return nil, err
	}
	return history, nil
}

func writeUploadHistory(history []uploadHistoryEntry) error {
	if err := os.MkdirAll(filepath.Dir(uploadHistoryPath()), 0700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(history, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(uploadHistoryPath(), data, 0600)
}

func beginUploadHistory(sessionID, targetPath string, files []string) (string, error) {
	uploadHistoryMu.Lock()
	defer uploadHistoryMu.Unlock()
	history, err := readUploadHistory()
	if err != nil {
		return "", err
	}
	id := fmt.Sprintf("%d", time.Now().UnixNano())
	entryFiles := append([]string(nil), files...)
	history = append([]uploadHistoryEntry{{
		ID:         id,
		SessionID:  sessionID,
		Files:      entryFiles,
		TargetPath: targetPath,
		Status:     "排队中",
		StartedAt:  time.Now().Unix(),
	}}, history...)
	if len(history) > 100 {
		history = history[:100]
	}
	return id, writeUploadHistory(history)
}

func updateUploadHistory(id, status string) {
	uploadHistoryMu.Lock()
	defer uploadHistoryMu.Unlock()
	history, err := readUploadHistory()
	if err != nil {
		return
	}
	for index := range history {
		if history[index].ID == id {
			history[index].Status = status
			break
		}
	}
	_ = writeUploadHistory(history)
}

func finishUploadHistory(id, status, uploadErr string) {
	uploadHistoryMu.Lock()
	defer uploadHistoryMu.Unlock()
	history, err := readUploadHistory()
	if err != nil {
		return
	}
	for index := range history {
		if history[index].ID != id {
			continue
		}
		history[index].Status = status
		history[index].Error = uploadErr
		history[index].FinishedAt = time.Now().Unix()
		break
	}
	_ = writeUploadHistory(history)
}

func listUploadHistory(sessionID string) ([]uploadHistoryEntry, error) {
	uploadHistoryMu.Lock()
	defer uploadHistoryMu.Unlock()
	history, err := readUploadHistory()
	if err != nil {
		return nil, err
	}
	filtered := make([]uploadHistoryEntry, 0, len(history))
	for _, entry := range history {
		if entry.SessionID == sessionID {
			filtered = append(filtered, entry)
		}
	}
	return filtered, nil
}
