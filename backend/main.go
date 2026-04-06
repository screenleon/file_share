package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"syscall"
	"time"
)

const uploadDir = "/app/uploads"

var safeFilenameRegex = regexp.MustCompile(`[^a-zA-Z0-9._-]`)

// Track files currently being uploaded — these cannot be downloaded
var (
	uploadingFiles = make(map[string]bool)
	uploadingMu    sync.RWMutex
)

// Track file metadata (uploader IP, download count)
type FileMeta struct {
	UploaderIP    string `json:"uploaderIp"`
	DownloadCount int    `json:"downloadCount"`
	Uploading     bool   `json:"uploading"`
}

var (
	fileMetas   = make(map[string]*FileMeta)
	fileMetasMu sync.RWMutex
)

func main() {
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		log.Fatalf("Failed to create upload directory: %v", err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/api/upload", corsMiddleware(handleUpload))
	mux.HandleFunc("/api/files", corsMiddleware(handleFiles))
	mux.HandleFunc("/api/download/", corsMiddleware(handleDownload))
	mux.HandleFunc("/api/files/", corsMiddleware(handleDeleteFile))

	server := &http.Server{
		Addr:              ":8080",
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
	}

	// Graceful shutdown with cleanup
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-quit
		log.Println("Shutting down server...")
		cleanup()
		if err := server.Close(); err != nil {
			log.Printf("Server close error: %v", err)
		}
	}()

	log.Printf("File share server starting on :8080")
	log.Printf("Upload directory: %s", uploadDir)
	if err := server.ListenAndServe(); err != http.ErrServerClosed {
		log.Fatalf("Server error: %v", err)
	}
	log.Println("Server stopped.")
}

func cleanup() {
	log.Println("Cleaning up uploaded files...")
	entries, err := os.ReadDir(uploadDir)
	if err != nil {
		log.Printf("Cleanup read error: %v", err)
		return
	}
	for _, entry := range entries {
		path := filepath.Join(uploadDir, entry.Name())
		if err := os.RemoveAll(path); err != nil {
			log.Printf("Failed to remove %s: %v", path, err)
		} else {
			log.Printf("Removed: %s", path)
		}
	}
	log.Println("Cleanup complete.")
}

func corsMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}
		next(w, r)
	}
}

func sanitizeFilename(name string) string {
	// Remove path separators to prevent traversal
	name = filepath.Base(name)
	// Replace unsafe characters
	name = safeFilenameRegex.ReplaceAllString(name, "_")
	if name == "" || name == "." || name == ".." {
		name = fmt.Sprintf("file_%d", time.Now().UnixNano())
	}
	return name
}

// handleUpload streams the uploaded file directly to disk
func handleUpload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	uploaderIP := getClientIP(r)

	// Use multipart reader for streaming — does NOT buffer entire file in memory
	reader, err := r.MultipartReader()
	if err != nil {
		http.Error(w, "Failed to read multipart form", http.StatusBadRequest)
		return
	}

	var uploaded []string

	for {
		part, err := reader.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			http.Error(w, "Error reading upload", http.StatusInternalServerError)
			return
		}

		if part.FormName() != "files" {
			part.Close()
			continue
		}

		filename := sanitizeFilename(part.FileName())
		destPath := filepath.Join(uploadDir, filename)

		// Ensure we don't escape the upload directory
		absPath, err := filepath.Abs(destPath)
		if err != nil || !strings.HasPrefix(absPath, uploadDir) {
			part.Close()
			http.Error(w, "Invalid filename", http.StatusBadRequest)
			return
		}

		// Handle duplicate filenames
		if _, err := os.Stat(destPath); err == nil {
			ext := filepath.Ext(filename)
			base := strings.TrimSuffix(filename, ext)
			filename = fmt.Sprintf("%s_%d%s", base, time.Now().UnixNano(), ext)
			destPath = filepath.Join(uploadDir, filename)
		}

		// Mark file as uploading — prevents download
		markUploading(filename, true, uploaderIP)

		dst, err := os.Create(destPath)
		if err != nil {
			markUploading(filename, false, "")
			part.Close()
			http.Error(w, "Failed to create file", http.StatusInternalServerError)
			return
		}

		// Stream directly from network to disk — constant memory usage
		written, err := io.Copy(dst, part)
		dst.Close()
		part.Close()

		if err != nil {
			markUploading(filename, false, "")
			os.Remove(destPath)
			http.Error(w, "Failed to save file", http.StatusInternalServerError)
			return
		}

		// Upload complete — unlock file for download
		markUploading(filename, false, uploaderIP)

		log.Printf("[UPLOAD] file=%s size=%d from=%s", filename, written, uploaderIP)
		uploaded = append(uploaded, filename)
	}

	if len(uploaded) == 0 {
		http.Error(w, "No files uploaded", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": fmt.Sprintf("Uploaded %d file(s)", len(uploaded)),
		"files":   uploaded,
	})
}

func markUploading(filename string, uploading bool, ip string) {
	uploadingMu.Lock()
	defer uploadingMu.Unlock()
	fileMetasMu.Lock()
	defer fileMetasMu.Unlock()

	if uploading {
		uploadingFiles[filename] = true
		fileMetas[filename] = &FileMeta{UploaderIP: ip, Uploading: true}
	} else {
		delete(uploadingFiles, filename)
		if meta, ok := fileMetas[filename]; ok {
			meta.Uploading = false
			if ip != "" {
				meta.UploaderIP = ip
			}
		} else if ip != "" {
			fileMetas[filename] = &FileMeta{UploaderIP: ip}
		}
	}
}

func isUploading(filename string) bool {
	uploadingMu.RLock()
	defer uploadingMu.RUnlock()
	return uploadingFiles[filename]
}

func getClientIP(r *http.Request) string {
	// Check X-Real-IP first (set by Nginx)
	if ip := r.Header.Get("X-Real-IP"); ip != "" {
		return ip
	}
	if ip := r.Header.Get("X-Forwarded-For"); ip != "" {
		return strings.Split(ip, ",")[0]
	}
	// Fallback to RemoteAddr
	ip := r.RemoteAddr
	if idx := strings.LastIndex(ip, ":"); idx != -1 {
		ip = ip[:idx]
	}
	return ip
}

type FileInfo struct {
	Name          string    `json:"name"`
	Size          int64     `json:"size"`
	ModTime       time.Time `json:"modTime"`
	Uploading     bool      `json:"uploading"`
	UploaderIP    string    `json:"uploaderIp"`
	DownloadCount int       `json:"downloadCount"`
}

func handleFiles(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	entries, err := os.ReadDir(uploadDir)
	if err != nil {
		http.Error(w, "Failed to list files", http.StatusInternalServerError)
		return
	}

	fileMetasMu.RLock()
	defer fileMetasMu.RUnlock()

	files := make([]FileInfo, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		fi := FileInfo{
			Name:    entry.Name(),
			Size:    info.Size(),
			ModTime: info.ModTime(),
		}
		if meta, ok := fileMetas[entry.Name()]; ok {
			fi.Uploading = meta.Uploading
			fi.UploaderIP = meta.UploaderIP
			fi.DownloadCount = meta.DownloadCount
		}
		files = append(files, fi)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(files)
}

// handleDownload streams the file from disk to the client
func handleDownload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	filename := sanitizeFilename(strings.TrimPrefix(r.URL.Path, "/api/download/"))

	// Block download if file is still being uploaded
	if isUploading(filename) {
		http.Error(w, "File is still being uploaded", http.StatusConflict)
		return
	}

	filePath := filepath.Join(uploadDir, filename)

	// Path traversal protection
	absPath, err := filepath.Abs(filePath)
	if err != nil || !strings.HasPrefix(absPath, uploadDir) {
		http.Error(w, "Invalid filename", http.StatusBadRequest)
		return
	}

	file, err := os.Open(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			http.Error(w, "File not found", http.StatusNotFound)
		} else {
			http.Error(w, "Failed to open file", http.StatusInternalServerError)
		}
		return
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		http.Error(w, "Failed to get file info", http.StatusInternalServerError)
		return
	}

	// Increment download count
	downloaderIP := getClientIP(r)
	fileMetasMu.Lock()
	if meta, ok := fileMetas[filename]; ok {
		meta.DownloadCount++
	} else {
		fileMetas[filename] = &FileMeta{DownloadCount: 1}
	}
	fileMetasMu.Unlock()

	log.Printf("[DOWNLOAD] file=%s size=%d by=%s", filename, stat.Size(), downloaderIP)

	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Length", fmt.Sprintf("%d", stat.Size()))

	// Stream from disk to network — constant memory
	http.ServeContent(w, r, filename, stat.ModTime(), file)
}

func handleDeleteFile(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	filename := sanitizeFilename(strings.TrimPrefix(r.URL.Path, "/api/files/"))
	filePath := filepath.Join(uploadDir, filename)

	absPath, err := filepath.Abs(filePath)
	if err != nil || !strings.HasPrefix(absPath, uploadDir) {
		http.Error(w, "Invalid filename", http.StatusBadRequest)
		return
	}

	if isUploading(filename) {
		http.Error(w, "Cannot delete file while uploading", http.StatusConflict)
		return
	}

	if err := os.Remove(filePath); err != nil {
		if os.IsNotExist(err) {
			http.Error(w, "File not found", http.StatusNotFound)
		} else {
			http.Error(w, "Failed to delete file", http.StatusInternalServerError)
		}
		return
	}

	// Clean up metadata
	fileMetasMu.Lock()
	delete(fileMetas, filename)
	fileMetasMu.Unlock()

	clientIP := getClientIP(r)
	log.Printf("[DELETE] file=%s by=%s", filename, clientIP)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"message": fmt.Sprintf("Deleted %s", filename),
	})
}
