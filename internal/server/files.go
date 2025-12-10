// Package server 文件管理模块
// 提供安全的文件浏览、编辑、删除等功能，支持路径安全检查和审计日志
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

// MountPoint 文件系统挂载点
// 在容器环境中，宿主机文件系统通常挂载到 /hostfs
// 如果 /hostfs 不存在，则使用根路径 /
var MountPoint = getMountPoint()

// getMountPoint 获取文件系统挂载点
// 检查 /hostfs 是否存在，如果不存在则使用根路径
func getMountPoint() string {
	if _, err := os.Stat("/hostfs"); err == nil {
		return "/hostfs"
	}
	return "/"
}

// BlockList 禁止访问的目录列表
// 包含系统关键目录，防止误操作导致系统损坏
var BlockList = []string{
	"/proc",  // 进程信息虚拟文件系统
	"/sys",   // 系统信息虚拟文件系统
	"/dev",   // 设备文件目录
	"/boot",  // 系统启动文件目录
}

// FileInfo 文件信息结构体
// 包含文件的基本属性信息，用于前端文件列表显示
type FileInfo struct {
	Name    string `json:"name"`     // 文件名
	Size    int64  `json:"size"`     // 文件大小（字节）
	Mode    string `json:"mode"`     // 文件权限模式
	ModTime string `json:"mod_time"` // 最后修改时间
	IsDir   bool   `json:"is_dir"`   // 是否为目录
	IsLink  bool   `json:"is_link"`  // 是否为符号链接
}

// FileResponse 文件操作响应结构体
// 统一的 API 响应格式，包含状态码、消息和数据
type FileResponse struct {
	Code int         `json:"code"`           // 状态码（200=成功，其他=错误）
	Msg  string      `json:"msg"`            // 响应消息
	Data interface{} `json:"data,omitempty"` // 响应数据（可选）
}

// jsonResponse 发送 JSON 格式的响应
// 统一处理 API 响应格式，设置正确的 Content-Type 头
func jsonResponse(w http.ResponseWriter, code int, msg string, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(FileResponse{
		Code: code,
		Msg:  msg,
		Data: data,
	})
}

// resolveSafePath 安全路径解析函数
// 防止路径遍历攻击和访问敏感系统目录
// 参数：userPath - 用户提供的路径
// 返回：安全的绝对路径和可能的错误
func resolveSafePath(userPath string) (string, error) {
	// 清理路径，移除 ".." 等危险元素
	cleanPath := filepath.Clean(userPath)
	
	// 如果用户路径是根路径，直接返回挂载点
	if cleanPath == "/" || cleanPath == "" {
		return MountPoint, nil
	}
	
	// 检查是否访问被禁止的系统目录
	for _, blocked := range BlockList {
		if strings.HasPrefix(cleanPath, blocked) {
			return "", fmt.Errorf("access denied: path '%s' is in blocklist", cleanPath)
		}
	}
	
	// 将用户路径映射到容器内的实际路径
	realPath := filepath.Join(MountPoint, cleanPath)
	
	// 防止路径逃逸攻击（确保路径在挂载点内）
	if !strings.HasPrefix(realPath, MountPoint) {
		return "", fmt.Errorf("access denied: path escape detected")
	}
	
	return realPath, nil
}

// handleFileList 处理文件列表请求
// 获取指定目录下的所有文件和子目录信息
// 支持安全路径检查和审计日志记录
func handleFileList(w http.ResponseWriter, r *http.Request) {
	// 获取用户请求的路径，默认为根目录
	userPath := r.URL.Query().Get("path")
	if userPath == "" { 
		userPath = "/" 
	}

	// 安全路径解析，防止路径遍历攻击
	realPath, err := resolveSafePath(userPath)
	if err != nil {
		logger.Info("[AUDIT] 🚨 非法访问尝试: %s | Error: %v", userPath, err)
		jsonResponse(w, 403, err.Error(), nil)
		return
	}

	// 读取目录内容
	entries, err := os.ReadDir(realPath)
	if err != nil {
		logger.Info("读取目录失败: %s | Error: %v", realPath, err)
		jsonResponse(w, 500, fmt.Sprintf("无法读取目录: %v", err), nil)
		return
	}

	// 构建文件信息列表
	files := make([]FileInfo, 0)
	
	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil { 
			continue // 跳过无法获取信息的文件
		}
		
		files = append(files, FileInfo{
			Name:    entry.Name(),
			Size:    info.Size(),
			Mode:    info.Mode().String(),
			ModTime: info.ModTime().Format("2006-01-02 15:04:05"),
			IsDir:   entry.IsDir(),
			IsLink:  info.Mode()&os.ModeSymlink != 0,
		})
	}

	// 排序：目录优先，然后按名称排序
	sort.Slice(files, func(i, j int) bool {
		if files[i].IsDir != files[j].IsDir {
			return files[i].IsDir // 目录排在前面
		}
		return files[i].Name < files[j].Name // 按名称字母顺序排序
	})

	// 返回文件列表
	jsonResponse(w, 200, "success", map[string]interface{}{
		"path":  userPath,
		"files": files,
	})
}

// handleFileContent 处理文件内容读取请求
// 读取指定文件的内容，支持文本文件的在线编辑
// 包含文件大小和格式检查，确保安全性
func handleFileContent(w http.ResponseWriter, r *http.Request) {
	// 获取文件路径
	userPath := r.URL.Query().Get("path")
	realPath, err := resolveSafePath(userPath)
	if err != nil {
		jsonResponse(w, 403, err.Error(), nil)
		return
	}

	// 检查文件是否存在并获取文件信息
	info, err := os.Stat(realPath)
	if err != nil {
		jsonResponse(w, 404, "文件不存在", nil)
		return
	}
	
	// 限制文件大小，防止内存溢出（最大 2MB）
	if info.Size() > 2*1024*1024 {
		jsonResponse(w, 400, "文件过大 (>2MB)，不支持在线编辑", nil)
		return
	}

	// 读取文件内容
	content, err := os.ReadFile(realPath)
	if err != nil {
		jsonResponse(w, 500, "读取失败", nil)
		return
	}

	// 检查是否为文本文件（UTF-8 编码）
	if !utf8.Valid(content) {
		jsonResponse(w, 400, "检测到二进制文件，不支持编辑", nil)
		return
	}

	// 直接返回文件内容（不使用 JSON 包装）
	w.Write(content)
}

// handleFileSave 处理文件保存请求
// 接收 JSON 格式的文件内容并安全地保存到指定路径
// 使用原子写入操作确保数据完整性
func handleFileSave(w http.ResponseWriter, r *http.Request) {
	// 只允许 POST 方法
	if r.Method != "POST" {
		jsonResponse(w, 405, "Method not allowed", nil)
		return
	}

	// 解析请求体中的 JSON 数据
	var req struct {
		Path    string `json:"path"`    // 文件路径
		Content string `json:"content"` // 文件内容
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonResponse(w, 400, "Invalid JSON", nil)
		return
	}

	// 安全路径解析
	realPath, err := resolveSafePath(req.Path)
	if err != nil {
		logger.Info("[AUDIT] 🚨 非法写入尝试: %s", req.Path)
		jsonResponse(w, 403, err.Error(), nil)
		return
	}

	// 使用原子写入操作保存文件
	if err := atomicWriteFile(realPath, []byte(req.Content), 0644); err != nil {
		logger.Info("[AUDIT] ❌ 文件保存失败: %s | Error: %v", req.Path, err)
		jsonResponse(w, 500, fmt.Sprintf("保存失败: %v", err), nil)
		return
	}

	// 记录审计日志
	logger.Info("[AUDIT] 📝 文件已修改: %s (Size: %d bytes)", req.Path, len(req.Content))
	jsonResponse(w, 200, "success", nil)
}

// handleFileAction 处理文件操作请求
// 支持删除文件/目录和创建目录操作
// 包含安全检查和审计日志记录
func handleFileAction(w http.ResponseWriter, r *http.Request) {
	// 获取操作类型和目标路径
	action := r.URL.Query().Get("type")
	userPath := r.URL.Query().Get("path")
	
	// 安全路径解析
	realPath, err := resolveSafePath(userPath)
	if err != nil {
		jsonResponse(w, 403, err.Error(), nil)
		return
	}

	// 根据操作类型执行相应操作
	switch action {
	case "delete":
		// 防止删除根目录的安全检查
		if userPath == "/" || realPath == MountPoint {
			jsonResponse(w, 403, "禁止删除根目录", nil)
			return
		}
		// 递归删除文件或目录
		err = os.RemoveAll(realPath)
		if err == nil {
			logger.Info("[AUDIT] 🗑️ 文件/目录已删除: %s", userPath)
		}
	case "mkdir":
		// 创建目录（包括父目录）
		err = os.MkdirAll(realPath, 0755)
		if err == nil {
			logger.Info("[AUDIT] 📂 目录已创建: %s", userPath)
		}
	default:
		// 不支持的操作类型
		jsonResponse(w, 400, "Unknown action", nil)
		return
	}

	// 检查操作结果
	if err != nil {
		jsonResponse(w, 500, fmt.Sprintf("操作失败: %v", err), nil)
		return
	}
	jsonResponse(w, 200, "success", nil)
}

// atomicWriteFile 原子写入文件函数
// 通过临时文件和重命名操作确保文件写入的原子性
// 防止写入过程中系统崩溃导致的文件损坏
// 参数：filename - 目标文件路径，data - 要写入的数据，perm - 文件权限
func atomicWriteFile(filename string, data []byte, perm os.FileMode) error {
	// 获取目标文件所在目录
	dir := filepath.Dir(filename)
	
	// 在同一目录下创建临时文件
	tmpFile, err := os.CreateTemp(dir, "qwq_tmp_*")
	if err != nil {
		return err
	}
	tmpName := tmpFile.Name()
	defer os.Remove(tmpName) // 确保临时文件被清理

	// 写入数据到临时文件
	if _, err := tmpFile.Write(data); err != nil {
		tmpFile.Close()
		return err
	}
	
	// 强制将数据刷新到磁盘
	if err := tmpFile.Sync(); err != nil {
		tmpFile.Close()
		return err
	}
	
	// 关闭临时文件
	if err := tmpFile.Close(); err != nil {
		return err
	}
	
	// 原子性地将临时文件重命名为目标文件
	// 这是整个操作的关键步骤，确保原子性
	return os.Rename(tmpName, filename)
}