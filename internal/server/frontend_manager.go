// Package server 前端资源管理模块
// 提供前端资源的验证、检查和管理功能
package server

import (
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"
)

// FrontendManager 前端资源管理器
// 负责前端资源的验证、检查和统计
// 支持 embed.FS 和 fs.FS 两种文件系统接口
type FrontendManager struct {
	embedFS embed.FS // 嵌入式文件系统（生产环境）
	fsFS    fs.FS    // 通用文件系统接口（测试环境）
	rootDir string   // 根目录路径
	useFS   bool     // 是否使用通用 fs.FS 接口
}

// ResourceStats 资源统计信息
type ResourceStats struct {
	TotalFiles     int      `json:"total_files"`     // 总文件数
	TotalSize      int64    `json:"total_size"`      // 总大小（字节）
	MissingFiles   []string `json:"missing_files"`   // 缺失的文件列表
	CorruptedFiles []string `json:"corrupted_files"` // 损坏的文件列表
	HasIndexHTML   bool     `json:"has_index_html"`  // 是否包含 index.html
	HasAssets      bool     `json:"has_assets"`      // 是否包含 assets 目录
	JSFiles        int      `json:"js_files"`        // JS 文件数量
	CSSFiles       int      `json:"css_files"`       // CSS 文件数量
	FontFiles      int      `json:"font_files"`      // 字体文件数量
}

// ResourceValidationResult 资源验证结果
type ResourceValidationResult struct {
	Valid       bool              `json:"valid"`        // 验证是否通过
	Errors      []string          `json:"errors"`       // 错误列表
	Warnings    []string          `json:"warnings"`     // 警告列表
	Stats       ResourceStats     `json:"stats"`        // 资源统计
	FileHashes  map[string]string `json:"file_hashes"`  // 文件哈希值
	Suggestions []string          `json:"suggestions"`  // 修复建议
}

// IsValid 实现 FrontendValidationResult 接口
func (r *ResourceValidationResult) IsValid() bool {
	return r.Valid
}

// GetErrors 实现 FrontendValidationResult 接口
func (r *ResourceValidationResult) GetErrors() []string {
	return r.Errors
}

// GetWarnings 实现 FrontendValidationResult 接口
func (r *ResourceValidationResult) GetWarnings() []string {
	return r.Warnings
}

// GetSuggestions 实现 FrontendValidationResult 接口
func (r *ResourceValidationResult) GetSuggestions() []string {
	return r.Suggestions
}

// RequiredFiles 必需的前端文件列表
var RequiredFiles = []string{
	"index.html",
}

// RequiredExtensions 必需的文件扩展名（至少需要一个）
var RequiredExtensions = map[string]string{
	".js":  "JavaScript 文件",
	".css": "CSS 样式文件",
}

// NewFrontendManager 创建前端资源管理器（使用 embed.FS）
func NewFrontendManager(embedFS embed.FS, rootDir string) *FrontendManager {
	return &FrontendManager{
		embedFS: embedFS,
		rootDir: rootDir,
		useFS:   false,
	}
}

// NewFrontendManagerWithFS 创建前端资源管理器（使用通用 fs.FS，用于测试）
func NewFrontendManagerWithFS(fsFS fs.FS, rootDir string) *FrontendManager {
	return &FrontendManager{
		fsFS:    fsFS,
		rootDir: rootDir,
		useFS:   true,
	}
}

// GetDefaultFrontendManager 获取默认的前端资源管理器
func GetDefaultFrontendManager() *FrontendManager {
	return NewFrontendManager(frontendDist, "dist")
}

// ValidateResources 验证前端资源完整性
// 检查所有必需的文件是否存在，并验证文件内容
func (fm *FrontendManager) ValidateResources() *ResourceValidationResult {
	result := &ResourceValidationResult{
		Valid:       true,
		Errors:      []string{},
		Warnings:    []string{},
		FileHashes:  make(map[string]string),
		Suggestions: []string{},
	}

	// 获取资源统计信息
	stats := fm.GetResourceStats()
	result.Stats = stats

	// 检查必需文件
	for _, file := range RequiredFiles {
		if !fm.FileExists(file) {
			result.Valid = false
			result.Errors = append(result.Errors, fmt.Sprintf("缺失必需文件: %s", file))
			result.Stats.MissingFiles = append(result.Stats.MissingFiles, file)
		}
	}

	// 检查是否有 JS 和 CSS 文件
	if stats.JSFiles == 0 {
		result.Valid = false
		result.Errors = append(result.Errors, "缺失 JavaScript 文件")
		result.Suggestions = append(result.Suggestions, "请运行 'cd frontend && npm run build' 重新构建前端")
	}

	if stats.CSSFiles == 0 {
		result.Warnings = append(result.Warnings, "缺失 CSS 样式文件")
	}

	// 检查 index.html 内容
	if fm.FileExists("index.html") {
		content, err := fm.ReadFile("index.html")
		if err != nil {
			result.Valid = false
			result.Errors = append(result.Errors, fmt.Sprintf("无法读取 index.html: %v", err))
		} else {
			// 验证 index.html 包含必要的元素
			if !strings.Contains(string(content), "<script") {
				result.Warnings = append(result.Warnings, "index.html 中未找到 script 标签")
			}
			if !strings.Contains(string(content), "<!DOCTYPE html>") && !strings.Contains(string(content), "<!doctype html>") {
				result.Warnings = append(result.Warnings, "index.html 缺少 DOCTYPE 声明")
			}
		}
	}

	// 计算文件哈希
	files, _ := fm.ListAllFiles()
	for _, file := range files {
		hash, err := fm.GetFileHash(file)
		if err == nil {
			result.FileHashes[file] = hash
		}
	}

	// 添加修复建议
	if !result.Valid {
		result.Suggestions = append(result.Suggestions,
			"1. 确保前端已正确构建: cd frontend && npm install && npm run build",
			"2. 确保构建产物已复制到 internal/server/dist 目录",
			"3. 重新编译 Go 程序以嵌入最新的前端资源",
		)
	}

	return result
}

// GetResourceStats 获取资源统计信息
func (fm *FrontendManager) GetResourceStats() ResourceStats {
	stats := ResourceStats{
		MissingFiles:   []string{},
		CorruptedFiles: []string{},
	}

	// 检查 index.html
	stats.HasIndexHTML = fm.FileExists("index.html")

	// 检查 assets 目录
	stats.HasAssets = fm.DirExists("assets")

	// 遍历所有文件
	files, err := fm.ListAllFiles()
	if err != nil {
		return stats
	}

	for _, file := range files {
		stats.TotalFiles++

		// 获取文件大小
		content, err := fm.ReadFile(file)
		if err != nil {
			stats.CorruptedFiles = append(stats.CorruptedFiles, file)
			continue
		}
		stats.TotalSize += int64(len(content))

		// 统计文件类型
		ext := strings.ToLower(filepath.Ext(file))
		switch ext {
		case ".js":
			stats.JSFiles++
		case ".css":
			stats.CSSFiles++
		case ".ttf", ".woff", ".woff2", ".eot":
			stats.FontFiles++
		}
	}

	return stats
}

// getFS 获取当前使用的文件系统接口
// 根据 useFS 标志返回通用 fs.FS 或嵌入式 embed.FS
func (fm *FrontendManager) getFS() fs.FS {
	if fm.useFS {
		return fm.fsFS
	}
	return fm.embedFS
}

// FileExists 检查文件是否存在
// 参数 path 为相对于 rootDir 的路径
func (fm *FrontendManager) FileExists(path string) bool {
	fullPath := filepath.Join(fm.rootDir, path)
	_, err := fs.Stat(fm.getFS(), fullPath)
	return err == nil
}

// DirExists 检查目录是否存在
// 参数 path 为相对于 rootDir 的路径
func (fm *FrontendManager) DirExists(path string) bool {
	fullPath := filepath.Join(fm.rootDir, path)
	info, err := fs.Stat(fm.getFS(), fullPath)
	if err != nil {
		return false
	}
	return info.IsDir()
}

// ReadFile 读取文件内容
// 参数 path 为相对于 rootDir 的路径，返回文件字节内容
func (fm *FrontendManager) ReadFile(path string) ([]byte, error) {
	fullPath := filepath.Join(fm.rootDir, path)
	return fs.ReadFile(fm.getFS(), fullPath)
}

// ListAllFiles 列出所有文件
// 递归遍历 rootDir 下的所有文件，返回相对路径列表
func (fm *FrontendManager) ListAllFiles() ([]string, error) {
	var files []string

	err := fs.WalkDir(fm.getFS(), fm.rootDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			// 移除根目录前缀
			relPath := strings.TrimPrefix(path, fm.rootDir+"/")
			if relPath == path {
				relPath = strings.TrimPrefix(path, fm.rootDir)
			}
			if relPath != "" {
				files = append(files, relPath)
			}
		}
		return nil
	})

	return files, err
}

// GetFileHash 获取文件的 SHA256 哈希值
func (fm *FrontendManager) GetFileHash(path string) (string, error) {
	content, err := fm.ReadFile(path)
	if err != nil {
		return "", err
	}

	hash := sha256.Sum256(content)
	return hex.EncodeToString(hash[:]), nil
}

// CheckEmbeddedResources 检查嵌入的资源
// 返回所有嵌入的文件列表
func (fm *FrontendManager) CheckEmbeddedResources() ([]string, error) {
	return fm.ListAllFiles()
}

// ValidateFileIntegrity 验证单个文件的完整性
// 检查文件是否可读且内容有效
func (fm *FrontendManager) ValidateFileIntegrity(path string) error {
	content, err := fm.ReadFile(path)
	if err != nil {
		return fmt.Errorf("无法读取文件 %s: %v", path, err)
	}

	if len(content) == 0 {
		return fmt.Errorf("文件 %s 内容为空", path)
	}

	// 根据文件类型进行额外验证
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".html":
		// HTML 文件应该包含基本的 HTML 结构
		contentStr := string(content)
		if !strings.Contains(contentStr, "<html") && !strings.Contains(contentStr, "<HTML") {
			return fmt.Errorf("文件 %s 不是有效的 HTML 文件", path)
		}
	case ".js":
		// JS 文件不应该以 HTML 错误页面开头
		if strings.HasPrefix(string(content), "<!DOCTYPE") || strings.HasPrefix(string(content), "<html") {
			return fmt.Errorf("文件 %s 内容异常，可能是错误页面", path)
		}
	case ".css":
		// CSS 文件不应该以 HTML 错误页面开头
		if strings.HasPrefix(string(content), "<!DOCTYPE") || strings.HasPrefix(string(content), "<html") {
			return fmt.Errorf("文件 %s 内容异常，可能是错误页面", path)
		}
	}

	return nil
}

// GetContentType 根据文件扩展名获取 Content-Type
func GetContentType(path string) string {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".html":
		return "text/html; charset=utf-8"
	case ".js":
		return "application/javascript; charset=utf-8"
	case ".css":
		return "text/css; charset=utf-8"
	case ".json":
		return "application/json; charset=utf-8"
	case ".png":
		return "image/png"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".svg":
		return "image/svg+xml"
	case ".ico":
		return "image/x-icon"
	case ".woff":
		return "font/woff"
	case ".woff2":
		return "font/woff2"
	case ".ttf":
		return "font/ttf"
	case ".eot":
		return "application/vnd.ms-fontobject"
	case ".map":
		return "application/json"
	default:
		return "application/octet-stream"
	}
}

// IsStaticResource 判断路径是否为静态资源
func IsStaticResource(path string) bool {
	staticPrefixes := []string{"assets/"}
	staticExtensions := []string{
		".js", ".css", ".png", ".jpg", ".jpeg", ".svg", ".json",
		".woff", ".woff2", ".ttf", ".eot", ".map", ".ico",
	}

	// 检查前缀
	for _, prefix := range staticPrefixes {
		if strings.HasPrefix(path, prefix) {
			return true
		}
	}

	// 检查扩展名
	ext := strings.ToLower(filepath.Ext(path))
	for _, staticExt := range staticExtensions {
		if ext == staticExt {
			return true
		}
	}

	return false
}

// FrontendManagerAdapter 前端管理器适配器
// 用于满足 deployment.FrontendManager 接口
type FrontendManagerAdapter struct {
	fm *FrontendManager
}

// NewFrontendManagerAdapter 创建前端管理器适配器
func NewFrontendManagerAdapter(fm *FrontendManager) *FrontendManagerAdapter {
	return &FrontendManagerAdapter{fm: fm}
}

// ValidateResources 实现 deployment.FrontendValidationResult 接口
func (a *FrontendManagerAdapter) ValidateResources() interface{} {
	return a.fm.ValidateResources()
}

// GetResourceStats 实现 deployment.FrontendManager 接口
func (a *FrontendManagerAdapter) GetResourceStats() interface{} {
	return a.fm.GetResourceStats()
}

// GetDefaultFrontendManagerAdapter 获取默认的前端管理器适配器
func GetDefaultFrontendManagerAdapter() *FrontendManagerAdapter {
	return NewFrontendManagerAdapter(GetDefaultFrontendManager())
}
