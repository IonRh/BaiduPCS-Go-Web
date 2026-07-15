package pcsweb

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"sync"
	"time"
)

const downloadTokenTTL = 24 * time.Hour

const (
	downloadTokenRemote = "remote"
	downloadTokenServer = "server"
)

type downloadToken struct {
	SessionID string
	Path      string
	Kind      string
	ExpiresAt time.Time
}

var downloadTokens = struct {
	sync.Mutex
	items map[string]downloadToken
}{
	items: make(map[string]downloadToken),
}

func createDownloadToken(sessionID, filePath, kind string) (string, error) {
	data := make([]byte, 32)
	if _, err := rand.Read(data); err != nil {
		return "", err
	}
	token := hex.EncodeToString(data)
	downloadTokens.Lock()
	now := time.Now()
	for existingToken, item := range downloadTokens.items {
		if now.After(item.ExpiresAt) {
			delete(downloadTokens.items, existingToken)
		}
	}
	downloadTokens.items[token] = downloadToken{
		SessionID: sessionID,
		Path:      filePath,
		Kind:      kind,
		ExpiresAt: now.Add(downloadTokenTTL),
	}
	downloadTokens.Unlock()
	return token, nil
}

func resolveDownloadToken(token, sessionID, kind string) (string, error) {
	downloadTokens.Lock()
	defer downloadTokens.Unlock()
	item, ok := downloadTokens.items[token]
	if !ok || time.Now().After(item.ExpiresAt) {
		delete(downloadTokens.items, token)
		return "", errors.New("download link is invalid or expired")
	}
	if item.SessionID != sessionID {
		return "", errors.New("download link does not belong to this session")
	}
	if item.Kind != kind {
		return "", errors.New("download link type is invalid")
	}
	return item.Path, nil
}
