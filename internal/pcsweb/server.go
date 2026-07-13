package pcsweb

import (
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/qjfoidnh/BaiduPCS-Go/baidupcs"
	"github.com/qjfoidnh/BaiduPCS-Go/internal/pcscommand"
	"github.com/qjfoidnh/BaiduPCS-Go/internal/pcsconfig"
)

//go:embed static/*
var staticFiles embed.FS

type Server struct {
	addr string
}

type fileItem struct {
	FsID     int64  `json:"fs_id"`
	Name     string `json:"name"`
	Path     string `json:"path"`
	Size     int64  `json:"size"`
	IsDir    bool   `json:"is_dir"`
	Modified int64  `json:"modified"`
}

type filesResponse struct {
	Path       string     `json:"path"`
	Items      []fileItem `json:"items"`
	Total      int        `json:"total"`
	Page       int        `json:"page"`
	PageSize   int        `json:"page_size"`
	TotalPages int        `json:"total_pages"`
}

type statusResponse struct {
	LoggedIn bool   `json:"logged_in"`
	UserName string `json:"user_name,omitempty"`
}

type renameRequest struct {
	From string `json:"from"`
	To   string `json:"to"`
}

// NewServer creates the local web server used by the CLI web command.
func NewServer(addr string) *Server {
	return &Server{addr: addr}
}

// Run starts the web server and blocks until it is stopped.
func (s *Server) Run() error {
	return http.ListenAndServe(s.addr, s.routes())
}

func (s *Server) routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/status", s.handleStatus)
	mux.HandleFunc("/api/login", s.handleLogin)
	mux.HandleFunc("/api/files", s.handleFiles)
	mux.HandleFunc("/api/mkdir", s.handleMkdir)
	mux.HandleFunc("/api/rename", s.handleRename)
	mux.HandleFunc("/api/upload", s.handleUpload)
	mux.HandleFunc("/api/file", s.handleFile)
	mux.HandleFunc("/api/download", s.handleDownload)

	staticFS, err := fsSub(staticFiles, "static")
	if err != nil {
		panic(err)
	}
	mux.Handle("/", http.FileServer(http.FS(staticFS)))
	return withNoCache(mux)
}

func fsSub(files embed.FS, dir string) (fs.FS, error) {
	sub, err := fs.Sub(files, dir)
	if err != nil {
		return nil, err
	}
	return sub, nil
}

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	user := pcsconfig.Config.ActiveUser()
	loggedIn := user.Name != "" && (user.BDUSS != "" || user.AccessToken != "" || user.COOKIES != "")
	writeJSON(w, http.StatusOK, statusResponse{LoggedIn: loggedIn, UserName: user.Name})
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}
	var request struct {
		Cookies string `json:"cookies"`
	}
	if !decodeJSON(w, r, &request) {
		return
	}
	request.Cookies = strings.TrimSpace(request.Cookies)
	if request.Cookies == "" {
		writeError(w, http.StatusBadRequest, errors.New("cookies are required"))
		return
	}
	if !hasCookie(request.Cookies, "BDUSS") {
		writeError(w, http.StatusBadRequest, errors.New("cookies must contain BDUSS"))
		return
	}

	user, err := pcsconfig.Config.SetupUserByBDUSS("", "", "", request.Cookies)
	if err != nil {
		writeError(w, http.StatusUnauthorized, err)
		return
	}
	if err := pcsconfig.Config.Save(); err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Errorf("login succeeded but saving config failed: %w", err))
		return
	}
	writeJSON(w, http.StatusOK, statusResponse{LoggedIn: true, UserName: user.Name})
}

func hasCookie(cookieHeader, name string) bool {
	pattern := regexp.MustCompile(`(?:^|;\s*)` + regexp.QuoteMeta(name) + `=[^;]+`)
	return pattern.MatchString(cookieHeader)
}

func (s *Server) handleFiles(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}

	pcs, err := activePCS()
	if err != nil {
		writeError(w, http.StatusUnauthorized, err)
		return
	}

	remotePath := normalizePath(r.URL.Query().Get("path"))
	items, pcsErr := pcs.FilesDirectoriesList(remotePath, baidupcs.DefaultOrderOptions)
	if pcsErr != nil {
		writeError(w, http.StatusBadGateway, pcsErr)
		return
	}

	page := queryInt(r, "page", 1)
	pageSize := queryInt(r, "page_size", 12)
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 12
	}
	if pageSize > 50 {
		pageSize = 50
	}
	total := len(items)
	totalPages := (total + pageSize - 1) / pageSize
	if totalPages == 0 {
		totalPages = 1
	}
	if page > totalPages {
		page = totalPages
	}
	start := (page - 1) * pageSize
	end := start + pageSize
	if end > total {
		end = total
	}
	pageItems := items
	if start < total {
		pageItems = items[start:end]
	} else {
		pageItems = nil
	}

	response := filesResponse{Path: remotePath, Items: make([]fileItem, 0, len(pageItems)), Total: total, Page: page, PageSize: pageSize, TotalPages: totalPages}
	for _, item := range pageItems {
		response.Items = append(response.Items, fileItem{
			FsID:     item.FsID,
			Name:     item.Filename,
			Path:     item.Path,
			Size:     item.Size,
			IsDir:    item.Isdir,
			Modified: item.Mtime,
		})
	}
	writeJSON(w, http.StatusOK, response)
}

func queryInt(r *http.Request, key string, fallback int) int {
	value, err := strconv.Atoi(r.URL.Query().Get(key))
	if err != nil {
		return fallback
	}
	return value
}

func (s *Server) handleMkdir(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}
	var request struct {
		Path string `json:"path"`
	}
	if !decodeJSON(w, r, &request) {
		return
	}

	pcs, err := activePCS()
	if err != nil {
		writeError(w, http.StatusUnauthorized, err)
		return
	}
	if request.Path == "" {
		writeError(w, http.StatusBadRequest, errors.New("path is required"))
		return
	}
	if pcsErr := pcs.Mkdir(normalizePath(request.Path)); pcsErr != nil {
		writeError(w, http.StatusBadGateway, pcsErr)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "folder created"})
}

func (s *Server) handleRename(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}
	var request renameRequest
	if !decodeJSON(w, r, &request) {
		return
	}
	if request.From == "" || request.To == "" {
		writeError(w, http.StatusBadRequest, errors.New("from and to are required"))
		return
	}

	pcs, err := activePCS()
	if err != nil {
		writeError(w, http.StatusUnauthorized, err)
		return
	}
	if pcsErr := pcs.Rename(normalizePath(request.From), normalizePath(request.To)); pcsErr != nil {
		writeError(w, http.StatusBadGateway, pcsErr)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "renamed"})
}

func (s *Server) handleUpload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}
	if _, err := activePCS(); err != nil {
		writeError(w, http.StatusUnauthorized, err)
		return
	}
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		writeError(w, http.StatusBadRequest, fmt.Errorf("invalid upload form: %w", err))
		return
	}
	targetPath := normalizePath(r.FormValue("target_path"))
	fileHeaders := r.MultipartForm.File["files"]
	if len(fileHeaders) == 0 {
		writeError(w, http.StatusBadRequest, errors.New("at least one file is required"))
		return
	}

	tempDir, err := os.MkdirTemp("", "baidupcs-web-upload-")
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	defer os.RemoveAll(tempDir)

	localPaths := make([]string, 0, len(fileHeaders))
	for index, header := range fileHeaders {
		name := filepath.Base(header.Filename)
		if name == "." || name == "" || name == string(filepath.Separator) {
			writeError(w, http.StatusBadRequest, errors.New("an uploaded file has an invalid name"))
			return
		}
		localPath := filepath.Join(tempDir, name)
		if _, statErr := os.Stat(localPath); statErr == nil {
			localPath = filepath.Join(tempDir, fmt.Sprintf("%d_%s", index, name))
		}

		source, openErr := header.Open()
		if openErr != nil {
			writeError(w, http.StatusBadRequest, openErr)
			return
		}
		destination, createErr := os.Create(localPath)
		if createErr == nil {
			_, createErr = io.Copy(destination, source)
			_ = destination.Close()
		}
		_ = source.Close()
		if createErr != nil {
			writeError(w, http.StatusInternalServerError, createErr)
			return
		}
		localPaths = append(localPaths, localPath)
	}

	// Reuse the existing upload pipeline so Web and CLI share upload behavior.
	pcscommand.RunUpload(localPaths, targetPath, nil)
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"message":     "upload finished",
		"target_path": targetPath,
		"count":       len(localPaths),
	})
}

func (s *Server) handleFile(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		methodNotAllowed(w)
		return
	}
	remotePath := r.URL.Query().Get("path")
	if remotePath == "" || normalizePath(remotePath) == "/" {
		writeError(w, http.StatusBadRequest, errors.New("a file path is required"))
		return
	}

	pcs, err := activePCS()
	if err != nil {
		writeError(w, http.StatusUnauthorized, err)
		return
	}
	if pcsErr := pcs.Remove(normalizePath(remotePath)); pcsErr != nil {
		writeError(w, http.StatusBadGateway, pcsErr)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "deleted"})
}

func (s *Server) handleDownload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	remotePath := r.URL.Query().Get("path")
	if remotePath == "" || normalizePath(remotePath) == "/" {
		writeError(w, http.StatusBadRequest, errors.New("a file path is required"))
		return
	}

	pcs, err := activePCS()
	if err != nil {
		writeError(w, http.StatusUnauthorized, err)
		return
	}
	remotePath = normalizePath(remotePath)
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", path.Base(remotePath)))
	err = pcs.DownloadFile(remotePath, func(downloadURL string, jar http.CookieJar) error {
		request, requestErr := http.NewRequestWithContext(r.Context(), http.MethodGet, downloadURL, nil)
		if requestErr != nil {
			return requestErr
		}
		request.Header.Set("User-Agent", pcsconfig.Config.PanUA)
		request.Header.Set("Referer", "https://pan.baidu.com/")
		request.Header.Set("Cookie", cookieHeader(jar, request.URL))
		response, responseErr := pcs.GetClient().Do(request)
		if responseErr != nil {
			return responseErr
		}
		defer response.Body.Close()
		if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
			return fmt.Errorf("download server returned %s", response.Status)
		}
		if contentType := response.Header.Get("Content-Type"); contentType != "" {
			w.Header().Set("Content-Type", contentType)
		}
		if response.ContentLength >= 0 {
			w.Header().Set("Content-Length", fmt.Sprint(response.ContentLength))
		}
		_, copyErr := io.Copy(w, response.Body)
		return copyErr
	})
	if err != nil {
		// Headers may already be sent for a partially streamed download.
		return
	}
}

func cookieHeader(jar http.CookieJar, requestURL *url.URL) string {
	cookies := jar.Cookies(requestURL)
	parts := make([]string, 0, len(cookies))
	for _, cookie := range cookies {
		parts = append(parts, cookie.Name+"="+cookie.Value)
	}
	return strings.Join(parts, "; ")
}

func activePCS() (*baidupcs.BaiduPCS, error) {
	user := pcsconfig.Config.ActiveUser()
	if user.Name == "" || (user.BDUSS == "" && user.AccessToken == "" && user.COOKIES == "") {
		return nil, errors.New("no active Baidu account; run BaiduPCS-Go login first")
	}
	return pcsconfig.Config.ActiveUserBaiduPCS(), nil
}

func normalizePath(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "/"
	}
	if !strings.HasPrefix(value, "/") {
		value = "/" + value
	}
	return path.Clean(value)
}

func writeJSON(w http.ResponseWriter, status int, value interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeError(w http.ResponseWriter, status int, err error) {
	writeJSON(w, status, map[string]string{"error": err.Error()})
}

func decodeJSON(w http.ResponseWriter, r *http.Request, target interface{}) bool {
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(target); err != nil {
		writeError(w, http.StatusBadRequest, fmt.Errorf("invalid JSON: %w", err))
		return false
	}
	return true
}

func methodNotAllowed(w http.ResponseWriter) {
	w.WriteHeader(http.StatusMethodNotAllowed)
}

func withNoCache(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		next.ServeHTTP(w, r)
	})
}
