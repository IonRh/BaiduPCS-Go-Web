package pcsweb

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"sync"
	"time"
)

const browserDownloadTokenTTL = 24 * time.Hour

type browserDownloadToken struct {
	SessionID  string
	RemotePath string
	ExpiresAt  time.Time
}

var browserDownloadTokens = struct {
	sync.Mutex
	items map[string]browserDownloadToken
}{
	items: make(map[string]browserDownloadToken),
}

func createBrowserDownloadToken(sessionID, remotePath string) (string, error) {
	data := make([]byte, 32)
	if _, err := rand.Read(data); err != nil {
		return "", err
	}
	token := hex.EncodeToString(data)
	browserDownloadTokens.Lock()
	now := time.Now()
	for existingToken, item := range browserDownloadTokens.items {
		if now.After(item.ExpiresAt) {
			delete(browserDownloadTokens.items, existingToken)
		}
	}
	browserDownloadTokens.items[token] = browserDownloadToken{
		SessionID:  sessionID,
		RemotePath: remotePath,
		ExpiresAt:  now.Add(browserDownloadTokenTTL),
	}
	browserDownloadTokens.Unlock()
	return token, nil
}

func resolveBrowserDownloadToken(token, sessionID string) (string, error) {
	browserDownloadTokens.Lock()
	defer browserDownloadTokens.Unlock()
	item, ok := browserDownloadTokens.items[token]
	if !ok || time.Now().After(item.ExpiresAt) {
		delete(browserDownloadTokens.items, token)
		return "", errors.New("download link is invalid or expired")
	}
	if item.SessionID != sessionID {
		return "", errors.New("download link does not belong to this session")
	}
	return item.RemotePath, nil
}
