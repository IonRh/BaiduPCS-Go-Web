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
	"strconv"
	"strings"
	"time"

	"github.com/qjfoidnh/BaiduPCS-Go/baidupcs"
	"github.com/qjfoidnh/BaiduPCS-Go/internal/pcscommand"
	"github.com/qjfoidnh/BaiduPCS-Go/internal/pcsconfig"
	"github.com/qjfoidnh/BaiduPCS-Go/internal/pcsfunctions/pcsdownload"
	"github.com/qjfoidnh/BaiduPCS-Go/internal/pcsfunctions/pcsupload"
	"github.com/qjfoidnh/BaiduPCS-Go/requester/downloader"
)

//go:embed static/*
var staticFiles embed.FS

type Server struct {
	addr           string
	accessPassword string
	access         *accessState
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

type uploadTaskResponse struct {
	ID       string `json:"id"`
	Path     string `json:"path"`
	Length   int64  `json:"length"`
	Uploaded int64  `json:"uploaded"`
	Progress int    `json:"progress"`
	Status   string `json:"status"`
}

type downloadTaskResponse struct {
	Path       string `json:"path"`
	SavePath   string `json:"save_path"`
	Total      int64  `json:"total"`
	Downloaded int64  `json:"downloaded"`
	Progress   int    `json:"progress"`
	Status     string `json:"status"`
}

type shareListItem struct {
	ShareID    int64  `json:"share_id"`
	Path       string `json:"path"`
	Link       string `json:"link"`
	Password   string `json:"password"`
	Visibility string `json:"visibility"`
	Expires    string `json:"expires"`
	ViewCount  int    `json:"view_count"`
}

type renameRequest struct {
	From string `json:"from"`
	To   string `json:"to"`
}

// NewServer creates the local web server used by the CLI web command.
func NewServer(addr string, accessPassword ...string) *Server {
	password := ""
	if len(accessPassword) > 0 {
		password = accessPassword[0]
	}
	return &Server{addr: addr, accessPassword: password, access: newAccessState()}
}

// Run starts the web server and blocks until it is stopped.
func (s *Server) Run() error {
	return http.ListenAndServe(s.addr, s.routes())
}

func (s *Server) routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/access/status", s.handleAccessStatus)
	mux.HandleFunc("/api/access/login", s.handleAccessLogin)
	mux.HandleFunc("/api/status", s.handleStatus)
	mux.HandleFunc("/api/login", s.handleLogin)
	mux.HandleFunc("/api/files", s.handleFiles)
	mux.HandleFunc("/api/upload/tasks", s.handleUploadTasks)
	mux.HandleFunc("/api/upload/history", s.handleUploadHistory)
	mux.HandleFunc("/api/upload/local", s.handleLocalUpload)
	mux.HandleFunc("/api/download/start", s.handleDownloadStart)
	mux.HandleFunc("/api/download/tasks", s.handleDownloadTasks)
	mux.HandleFunc("/api/download/history", s.handleDownloadHistory)
	mux.HandleFunc("/api/shares", s.handleShares)
	mux.HandleFunc("/api/shares/create", s.handleShareCreate)
	mux.HandleFunc("/api/shares/cancel", s.handleShareCancel)
	mux.HandleFunc("/api/mkdir", s.handleMkdir)
	mux.HandleFunc("/api/rename", s.handleRename)
	mux.HandleFunc("/api/upload", s.handleUpload)
	mux.HandleFunc("/api/file", s.handleFile)
	mux.HandleFunc("/api/download", s.handleDownload)
	mux.HandleFunc("/api/server-download", s.handleServerDownload)

	staticFS, err := fsSub(staticFiles, "static")
	if err != nil {
		panic(err)
	}
	mux.Handle("/", http.FileServer(http.FS(staticFS)))
	return withNoCache(s.withAccess(mux))
}

func fsSub(files embed.FS, dir string) (fs.FS, error) {
	sub, err := fs.Sub(files, dir)
	if err != nil {
		return nil, err
	}
	return sub, nil
}

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	ensureSession(w, r)
	user := pcsconfig.Config.ActiveUser()
	loggedIn := user.BDUSS != "" || user.AccessToken != "" || user.COOKIES != ""
	writeJSON(w, http.StatusOK, statusResponse{LoggedIn: loggedIn, UserName: displayUserName(user)})
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}
	ensureSession(w, r)
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
	bduss := cookieValue(request.Cookies, "BDUSS")
	if bduss == "" {
		writeError(w, http.StatusBadRequest, errors.New("cookies must contain BDUSS"))
		return
	}

	user, err := pcsconfig.Config.SetupUserByBDUSS(bduss, "", "", request.Cookies)
	if err != nil {
		writeError(w, http.StatusUnauthorized, err)
		return
	}
	if err := pcsconfig.Config.Save(); err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Errorf("login succeeded but saving config failed: %w", err))
		return
	}
	writeJSON(w, http.StatusOK, statusResponse{LoggedIn: true, UserName: displayUserName(user)})
}

func hasCookie(cookieHeader, name string) bool {
	return cookieValue(cookieHeader, name) != ""
}

func cookieValue(cookieHeader, name string) string {
	for _, part := range strings.Split(cookieHeader, ";") {
		key, value, ok := strings.Cut(strings.TrimSpace(part), "=")
		if ok && strings.TrimSpace(key) == name {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func displayUserName(user *pcsconfig.Baidu) string {
	if user == nil {
		return ""
	}
	if user.Name != "" {
		return user.Name
	}
	if user.UID != 0 {
		return fmt.Sprintf("UID %d", user.UID)
	}
	return ""
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

func (s *Server) handleUploadTasks(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	if _, err := activePCS(); err != nil {
		writeError(w, http.StatusUnauthorized, err)
		return
	}
	sessionID := ensureSession(w, r)
	webTasks := listUploadTasks(sessionID)
	tasks := make([]uploadTaskResponse, 0, len(webTasks))
	database, databaseErr := pcsupload.NewUploadingDatabase()
	if databaseErr == nil {
		defer database.Close()
	}
	for _, webTask := range webTasks {
		var uploaded int64
		if databaseErr == nil {
			for _, task := range database.UploadingList {
				if task == nil || task.LocalFileMeta == nil || task.Path != webTask.Path {
					continue
				}
				if task.State != nil {
					for _, block := range task.State.BlockList {
						if block != nil && block.CheckSum != "" {
							uploaded += block.Range.End - block.Range.Begin
						}
					}
				}
				break
			}
		}
		progress := 0
		if webTask.Length > 0 {
			progress = int(uploaded * 100 / webTask.Length)
			if progress > 100 {
				progress = 100
			}
		}
		status := webTask.Status
		if uploaded > 0 {
			status = "正在上传"
		}
		tasks = append(tasks, uploadTaskResponse{ID: webTask.ID, Path: webTask.Path, Length: webTask.Length, Uploaded: uploaded, Progress: progress, Status: status})
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"tasks": tasks})
}

func (s *Server) handleUploadHistory(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	if _, err := activePCS(); err != nil {
		writeError(w, http.StatusUnauthorized, err)
		return
	}
	history, err := listUploadHistory()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	page, pageSize := historyPageQuery(r)
	items, totalPages := paginateHistory(history, page, pageSize)
	if page > totalPages {
		page = totalPages
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"history": items, "total": len(history), "page": page, "page_size": pageSize, "total_pages": totalPages})
}

func (s *Server) handleDownloadStart(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}
	if _, err := activePCS(); err != nil {
		writeError(w, http.StatusUnauthorized, err)
		return
	}
	var request struct {
		Path string `json:"path"`
	}
	if !decodeJSON(w, r, &request) {
		return
	}
	request.Path = normalizePath(request.Path)
	if request.Path == "/" {
		writeError(w, http.StatusBadRequest, errors.New("a file or directory path is required"))
		return
	}
	sessionID := ensureSession(w, r)
	downloadRoot := filepath.Join(pcsconfig.Config.SaveDir, "WebDownloads", sessionID)
	savePath := filepath.Join(downloadRoot, filepath.FromSlash(strings.TrimPrefix(request.Path, "/")))
	if downloadPathActive(savePath) {
		writeError(w, http.StatusConflict, errors.New("this download is already running"))
		return
	}
	historyID, err := beginDownloadHistory(sessionID, request.Path, savePath)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	taskID := addDownloadTask(sessionID, request.Path, savePath)
	go func() {
		status := "已完成"
		downloadErr := ""
		defer func() {
			if recovered := recover(); recovered != nil {
				status = "失败"
				downloadErr = fmt.Sprint(recovered)
			}
			finishDownloadHistory(historyID, status, downloadErr)
			removeDownloadTask(taskID)
		}()
		pcscommand.RunDownload([]string{request.Path}, &pcscommand.DownloadOptions{SaveTo: downloadRoot})
	}()
	writeJSON(w, http.StatusAccepted, map[string]string{"message": "download queued", "path": request.Path})
}

func (s *Server) handleDownloadHistory(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	if _, err := activePCS(); err != nil {
		writeError(w, http.StatusUnauthorized, err)
		return
	}
	history, err := listDownloadHistory()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	page, pageSize := historyPageQuery(r)
	items, totalPages := paginateHistory(history, page, pageSize)
	if page > totalPages {
		page = totalPages
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"history": items, "total": len(history), "page": page, "page_size": pageSize, "total_pages": totalPages})
}

func (s *Server) handleDownloadTasks(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	if _, err := activePCS(); err != nil {
		writeError(w, http.StatusUnauthorized, err)
		return
	}
	sessionID := ensureSession(w, r)
	webTasks := listDownloadTasks(sessionID)
	tasks := make([]downloadTaskResponse, 0, len(webTasks))
	for _, webTask := range webTasks {
		total, downloaded := downloadProgress(webTask.SavePath)
		progress := 0
		if total > 0 {
			progress = int(downloaded * 100 / total)
			if progress > 100 {
				progress = 100
			}
		}
		tasks = append(tasks, downloadTaskResponse{Path: webTask.Path, SavePath: webTask.SavePath, Total: total, Downloaded: downloaded, Progress: progress, Status: webTask.Status})
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"tasks": tasks})
}

func (s *Server) handleShareCreate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}
	pcs, err := activePCS()
	if err != nil {
		writeError(w, http.StatusUnauthorized, err)
		return
	}
	var request struct {
		Paths    []string `json:"paths"`
		Password string   `json:"password"`
		Period   int      `json:"period"`
	}
	if !decodeJSON(w, r, &request) {
		return
	}
	paths := make([]string, 0, len(request.Paths))
	for _, rawPath := range request.Paths {
		for _, line := range strings.Split(rawPath, "\n") {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			paths = append(paths, normalizePath(line))
		}
	}
	if len(paths) == 0 {
		writeError(w, http.StatusBadRequest, errors.New("at least one file or directory path is required"))
		return
	}
	if request.Period < 0 {
		writeError(w, http.StatusBadRequest, errors.New("period must be zero or a positive number of days"))
		return
	}
	request.Password = strings.TrimSpace(request.Password)
	if request.Password != "" && len(request.Password) != 4 {
		writeError(w, http.StatusBadRequest, errors.New("the share password must contain 4 characters"))
		return
	}
	shared, pcsErr := pcs.ShareSet(paths, &baidupcs.ShareOption{Password: request.Password, Period: request.Period})
	if pcsErr != nil {
		writeError(w, http.StatusBadGateway, pcsErr)
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"share_id": shared.ShareID,
		"link":     shared.Link,
		"password": shared.Pwd,
	})
}

func (s *Server) handleShares(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	pcs, err := activePCS()
	if err != nil {
		writeError(w, http.StatusUnauthorized, err)
		return
	}
	page := queryInt(r, "page", 1)
	if page < 1 {
		page = 1
	}
	records, pcsErr := pcs.ShareList(page)
	if pcsErr != nil {
		writeError(w, http.StatusBadGateway, pcsErr)
		return
	}
	items := make([]shareListItem, 0, len(records))
	for _, record := range records {
		if record == nil {
			continue
		}
		link := record.Shortlink
		password := ""
		if record.Public == 0 && record.ExpireType != -1 {
			if info, infoErr := pcs.ShareSURLInfo(record.ShareID); infoErr == nil && info != nil {
				password = strings.TrimSpace(info.Pwd)
				if link == "" {
					link = info.ShortURL
				}
			}
		}
		visibility := "私密分享"
		if record.Public != 0 {
			visibility = "公开分享"
		}
		items = append(items, shareListItem{
			ShareID:    record.ShareID,
			Path:       record.TypicalPath,
			Link:       link,
			Password:   password,
			Visibility: visibility,
			Expires:    shareExpiryLabel(record),
			ViewCount:  record.ViewCount,
		})
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"shares":   items,
		"page":     page,
		"has_next": len(items) > 0,
	})
}

func (s *Server) handleShareCancel(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}
	pcs, err := activePCS()
	if err != nil {
		writeError(w, http.StatusUnauthorized, err)
		return
	}
	var request struct {
		ShareIDs []int64 `json:"share_ids"`
	}
	if !decodeJSON(w, r, &request) {
		return
	}
	if len(request.ShareIDs) == 0 {
		writeError(w, http.StatusBadRequest, errors.New("at least one share id is required"))
		return
	}
	if pcsErr := pcs.ShareCancel(request.ShareIDs); pcsErr != nil {
		writeError(w, http.StatusBadGateway, pcsErr)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "share cancelled"})
}

func shareExpiryLabel(record *baidupcs.ShareRecordInfo) string {
	if record.ExpireType == -1 {
		return "已过期"
	}
	if record.ExpireTime == 0 {
		return "永久"
	}
	return time.Now().Add(time.Duration(record.ExpireTime) * time.Second).Format("2006-01-02 15:04")
}

func downloadProgress(savePath string) (total, downloaded int64) {
	stateFile, err := os.Open(savePath + pcsdownload.DownloadSuffix)
	if err != nil {
		return
	}
	defer stateFile.Close()
	state := downloader.NewInstanceState(stateFile, downloader.InstanceStateStorageFormatProto3)
	info := state.Get()
	_ = state.Close()
	if info == nil || info.DownloadStatus == nil {
		return
	}
	return info.DownloadStatus.TotalSize(), info.DownloadStatus.Downloaded()
}

func queryInt(r *http.Request, key string, fallback int) int {
	value, err := strconv.Atoi(r.URL.Query().Get(key))
	if err != nil {
		return fallback
	}
	return value
}

func historyPageQuery(r *http.Request) (page, pageSize int) {
	page = queryInt(r, "page", 1)
	pageSize = queryInt(r, "page_size", 10)
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 10
	}
	if pageSize > 50 {
		pageSize = 50
	}
	return page, pageSize
}

func paginateHistory[T any](history []T, page, pageSize int) ([]T, int) {
	totalPages := (len(history) + pageSize - 1) / pageSize
	if totalPages == 0 {
		totalPages = 1
	}
	if page > totalPages {
		page = totalPages
	}
	start := (page - 1) * pageSize
	if start >= len(history) {
		return []T{}, totalPages
	}
	end := start + pageSize
	if end > len(history) {
		end = len(history)
	}
	return history[start:end], totalPages
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
	sessionID := ensureSession(w, r)
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
	keepTempDir := true
	defer func() {
		if keepTempDir {
			_ = os.RemoveAll(tempDir)
		}
	}()

	localPaths := make([]string, 0, len(fileHeaders))
	fileNames := make([]string, 0, len(fileHeaders))
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
		fileNames = append(fileNames, name)
	}

	keepTempDir = false
	go func() {
		defer os.RemoveAll(tempDir)
		_ = runUploadJob(sessionID, targetPath, localPaths, fileNames)
	}()
	writeJSON(w, http.StatusAccepted, map[string]interface{}{
		"message":     "upload queued",
		"target_path": targetPath,
		"count":       len(localPaths),
	})
}

func (s *Server) handleLocalUpload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}
	if _, err := activePCS(); err != nil {
		writeError(w, http.StatusUnauthorized, err)
		return
	}
	sessionID := ensureSession(w, r)
	var request struct {
		LocalPath  string `json:"local_path"`
		TargetPath string `json:"target_path"`
	}
	if !decodeJSON(w, r, &request) {
		return
	}
	localPath := filepath.Clean(strings.TrimSpace(request.LocalPath))
	if localPath == "." || localPath == "" {
		writeError(w, http.StatusBadRequest, errors.New("local_path is required"))
		return
	}
	info, err := os.Stat(localPath)
	if err != nil {
		writeError(w, http.StatusBadRequest, fmt.Errorf("server path is not accessible: %w", err))
		return
	}
	name := filepath.Base(localPath)
	if info.IsDir() {
		name += " (目录)"
	}
	targetPath := normalizePath(request.TargetPath)
	go func() {
		_ = runUploadJob(sessionID, targetPath, []string{localPath}, []string{name})
	}()
	writeJSON(w, http.StatusAccepted, map[string]string{"message": "upload queued", "path": localPath})
}

func runUploadJob(sessionID, targetPath string, localPaths, fileNames []string) (err error) {
	historyID, err := beginUploadHistory(sessionID, targetPath, fileNames)
	if err != nil {
		return err
	}
	taskIDs := make([]string, 0, len(localPaths))
	for _, localPath := range localPaths {
		var length int64
		if info, statErr := os.Stat(localPath); statErr == nil {
			length = info.Size()
		}
		taskIDs = append(taskIDs, addUploadTask(sessionID, localPath, length))
	}
	uploadExecutionMu.Lock()
	defer uploadExecutionMu.Unlock()
	defer removeUploadTasks(taskIDs)
	status := "已完成"
	uploadErr := ""
	defer func() {
		if recovered := recover(); recovered != nil {
			status = "失败"
			uploadErr = fmt.Sprint(recovered)
		}
		finishUploadHistory(historyID, status, uploadErr)
	}()
	updateUploadHistory(historyID, "上传中")
	for _, taskID := range taskIDs {
		updateUploadTask(taskID, "上传中")
	}
	pcscommand.RunUpload(localPaths, targetPath, nil)
	return nil
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
	sessionID := ensureSession(w, r)
	historyID, historyErr := beginDownloadHistory(sessionID, remotePath, "浏览器下载")
	err = pcs.DownloadFile(remotePath, func(downloadURL string, jar http.CookieJar) error {
		request, requestErr := http.NewRequestWithContext(r.Context(), http.MethodGet, downloadURL, nil)
		if requestErr != nil {
			return requestErr
		}
		request.Header.Set("User-Agent", pcsconfig.Config.PanUA)
		request.Header.Set("Referer", "https://pan.baidu.com/")
		request.Header.Set("Cookie", cookieHeader(jar, request.URL))
		if rangeHeader := r.Header.Get("Range"); rangeHeader != "" {
			request.Header.Set("Range", rangeHeader)
		}
		if ifRange := r.Header.Get("If-Range"); ifRange != "" {
			request.Header.Set("If-Range", ifRange)
		}
		response, responseErr := pcs.GetClient().Do(request)
		if responseErr != nil {
			return responseErr
		}
		defer response.Body.Close()
		if response.StatusCode == http.StatusRequestedRangeNotSatisfiable {
			if contentRange := response.Header.Get("Content-Range"); contentRange != "" {
				w.Header().Set("Content-Range", contentRange)
			}
			w.WriteHeader(response.StatusCode)
			return nil
		}
		if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
			return fmt.Errorf("download server returned %s", response.Status)
		}
		for _, headerName := range []string{"Content-Type", "Content-Length", "Content-Range", "Accept-Ranges", "ETag", "Last-Modified"} {
			if value := response.Header.Get(headerName); value != "" {
				w.Header().Set(headerName, value)
			}
		}
		w.WriteHeader(response.StatusCode)
		_, copyErr := io.Copy(w, response.Body)
		return copyErr
	})
	if err != nil {
		if historyErr == nil {
			finishDownloadHistory(historyID, "失败", err.Error())
		}
		// Headers may already be sent for a partially streamed download.
		return
	}
	if historyErr == nil {
		finishDownloadHistory(historyID, "已完成", "")
	}
}

func (s *Server) handleServerDownload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	localPath := strings.TrimSpace(r.URL.Query().Get("path"))
	if localPath == "" {
		writeError(w, http.StatusBadRequest, errors.New("a server file path is required"))
		return
	}
	info, err := os.Stat(localPath)
	if err != nil {
		writeError(w, http.StatusBadRequest, fmt.Errorf("server file is not accessible: %w", err))
		return
	}
	if !info.Mode().IsRegular() {
		writeError(w, http.StatusBadRequest, errors.New("only a server file can be downloaded"))
		return
	}
	file, err := os.Open(localPath)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	defer file.Close()

	sessionID := ensureSession(w, r)
	historyID, historyErr := beginServerDownloadHistory(sessionID, localPath, info.Size(), info.ModTime())
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", info.Name()))
	http.ServeContent(w, r, info.Name(), info.ModTime(), file)
	if historyErr == nil {
		finishDownloadHistory(historyID, "已完成", "")
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
	if user.BDUSS == "" && user.AccessToken == "" && user.COOKIES == "" {
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
