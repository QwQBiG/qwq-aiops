package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"qwq/internal/logger"
	"sort"
	"strings"
	"unicode/utf8"
)

const MountPoint = "/hostfs"

var BlockList = []string{
	"/proc",
	"/sys",
	"/dev",
	"/boot",
}

type FileInfo struct {
	Name    string `json:"name"`
	Size    int64  `json:"size"`
	Mode    string `json:"mode"`
	ModTime string `json:"mod_time"`
	IsDir   bool   `json:"is_dir"`
	IsLink  bool   `json:"is_link"`
}

type FileResponse struct {
	Code int         `json:"code"`
	Msg  string      `json:"msg"`
	Data interface{} `json:"data,omitempty"`
}

func jsonResponse(w http.ResponseWriter, code int, msg string, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(FileResponse{
		Code: code,
		Msg:  msg,
		Data: data,
	})
}

func resolveSafePath(userPath string) (string, error) {
	cleanPath := filepath.Clean(userPath)
	for _, blocked := range BlockList {
		if strings.HasPrefix(cleanPath, blocked) {
			return "", fmt.Errorf("access denied: path '%s' is in blocklist", cleanPath)
		}
	}
	realPath := filepath.Join(MountPoint, cleanPath)
	if !strings.HasPrefix(realPath, MountPoint) {
		return "", fmt.Errorf("access denied: path escape detected")
	}
	return realPath, nil
}

func handleFileList(w http.ResponseWriter, r *http.Request) {
	userPath := r.URL.Query().Get("path")
	if userPath == "" { userPath = "/" }

	realPath, err := resolveSafePath(userPath)
	if err != nil {
		logger.Info("[AUDIT] 🚨 非法访问尝试: %s | Error: %v", userPath, err)
		jsonResponse(w, 403, err.Error(), nil)
		return
	}

	entries, err := os.ReadDir(realPath)
	if err != nil {
		logger.Info("读取目录失败: %s | Error: %v", realPath, err)
		jsonResponse(w, 500, fmt.Sprintf("无法读取目录: %v", err), nil)
		return
	}

	files := make([]FileInfo, 0)
	
	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil { continue }
		
		files = append(files, FileInfo{
			Name:    entry.Name(),
			Size:    info.Size(),
			Mode:    info.Mode().String(),
			ModTime: info.ModTime().Format("2006-01-02 15:04:05"),
			IsDir:   entry.IsDir(),
			IsLink:  info.Mode()&os.ModeSymlink != 0,
		})
	}

	sort.Slice(files, func(i, j int) bool {
		if files[i].IsDir != files[j].IsDir {
			return files[i].IsDir
		}
		return files[i].Name < files[j].Name
	})

	jsonResponse(w, 200, "success", map[string]interface{}{
		"path":  userPath,
		"files": files,
	})
}

func handleFileContent(w http.ResponseWriter, r *http.Request) {
	userPath := r.URL.Query().Get("path")
	realPath, err := resolveSafePath(userPath)
	if err != nil {
		jsonResponse(w, 403, err.Error(), nil)
		return
	}

	info, err := os.Stat(realPath)
	if err != nil {
		jsonResponse(w, 404, "文件不存在", nil)
		return
	}
	if info.Size() > 2*1024*1024 {
		jsonResponse(w, 400, "文件过大 (>2MB)，不支持在线编辑", nil)
		return
	}

	content, err := os.ReadFile(realPath)
	if err != nil {
		jsonResponse(w, 500, "读取失败", nil)
		return
	}

	if !utf8.Valid(content) {
		jsonResponse(w, 400, "检测到二进制文件，不支持编辑", nil)
		return
	}

	w.Write(content)
}

func handleFileSave(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		jsonResponse(w, 405, "Method not allowed", nil)
		return
	}

	var req struct {
		Path    string `json:"path"`
		Content string `json:"content"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonResponse(w, 400, "Invalid JSON", nil)
		return
	}

	realPath, err := resolveSafePath(req.Path)
	if err != nil {
		logger.Info("[AUDIT] 🚨 非法写入尝试: %s", req.Path)
		jsonResponse(w, 403, err.Error(), nil)
		return
	}

	if err := atomicWriteFile(realPath, []byte(req.Content), 0644); err != nil {
		logger.Info("[AUDIT] ❌ 文件保存失败: %s | Error: %v", req.Path, err)
		jsonResponse(w, 500, fmt.Sprintf("保存失败: %v", err), nil)
		return
	}

	logger.Info("[AUDIT] 📝 文件已修改: %s (Size: %d bytes)", req.Path, len(req.Content))
	jsonResponse(w, 200, "success", nil)
}

func handleFileAction(w http.ResponseWriter, r *http.Request) {
	action := r.URL.Query().Get("type")
	userPath := r.URL.Query().Get("path")
	
	realPath, err := resolveSafePath(userPath)
	if err != nil {
		jsonResponse(w, 403, err.Error(), nil)
		return
	}

	switch action {
	case "delete":
		if userPath == "/" || realPath == MountPoint {
			jsonResponse(w, 403, "禁止删除根目录", nil)
			return
		}
		err = os.RemoveAll(realPath)
		if err == nil {
			logger.Info("[AUDIT] 🗑️ 文件/目录已删除: %s", userPath)
		}
	case "mkdir":
		err = os.MkdirAll(realPath, 0755)
		if err == nil {
			logger.Info("[AUDIT] 📂 目录已创建: %s", userPath)
		}
	default:
		jsonResponse(w, 400, "Unknown action", nil)
		return
	}

	if err != nil {
		jsonResponse(w, 500, fmt.Sprintf("操作失败: %v", err), nil)
		return
	}
	jsonResponse(w, 200, "success", nil)
}

func atomicWriteFile(filename string, data []byte, perm os.FileMode) error {
	dir := filepath.Dir(filename)
	tmpFile, err := os.CreateTemp(dir, "qwq_tmp_*")
	if err != nil {
		return err
	}
	tmpName := tmpFile.Name()
	defer os.Remove(tmpName)

	if _, err := tmpFile.Write(data); err != nil {
		tmpFile.Close()
		return err
	}
	if err := tmpFile.Sync(); err != nil {
		tmpFile.Close()
		return err
	}
	if err := tmpFile.Close(); err != nil {
		return err
	}
	return os.Rename(tmpName, filename)
}