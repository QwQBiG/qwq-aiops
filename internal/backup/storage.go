package backup

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// StorageBackend 存储后端接口
type StorageBackend interface {
	// Upload 上传文件到存储后端
	Upload(ctx context.Context, localPath string, config map[string]interface{}) (string, error)
	
	// Download 从存储后端下载文件
	Download(ctx context.Context, remotePath, localPath string, config map[string]interface{}) error
	
	// Delete 删除存储后端的文件
	Delete(ctx context.Context, remotePath string, config map[string]interface{}) error
	
	// Exists 检查文件是否存在
	Exists(ctx context.Context, remotePath string, config map[string]interface{}) (bool, error)
	
	// List 列出存储后端的文件
	List(ctx context.Context, prefix string, config map[string]interface{}) ([]string, error)
}

// LocalStorage 本地存储实现
type LocalStorage struct {
	basePath string
}

// NewLocalStorage 创建本地存储
func NewLocalStorage() *LocalStorage {
	return &LocalStorage{
		basePath: "/var/backups/qwq",
	}
}

// Upload 上传文件到本地存储
func (s *LocalStorage) Upload(ctx context.Context, localPath string, config map[string]interface{}) (string, error) {
	// 从配置中获取目标路径
	targetDir := s.basePath
	if path, ok := config["path"].(string); ok && path != "" {
		targetDir = path
	}
	
	// 确保目标目录存在
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create target directory: %w", err)
	}
	
	// 生成目标文件路径
	filename := filepath.Base(localPath)
	targetPath := filepath.Join(targetDir, filename)
	
	// 复制文件
	if err := copyFile(localPath, targetPath); err != nil {
		return "", fmt.Errorf("failed to copy file: %w", err)
	}
	
	return targetPath, nil
}

// Download 从本地存储下载文件
func (s *LocalStorage) Download(ctx context.Context, remotePath, localPath string, config map[string]interface{}) error {
	return copyFile(remotePath, localPath)
}

// Delete 删除本地存储的文件
func (s *LocalStorage) Delete(ctx context.Context, remotePath string, config map[string]interface{}) error {
	return os.Remove(remotePath)
}

// Exists 检查文件是否存在
func (s *LocalStorage) Exists(ctx context.Context, remotePath string, config map[string]interface{}) (bool, error) {
	_, err := os.Stat(remotePath)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

// List 列出本地存储的文件
func (s *LocalStorage) List(ctx context.Context, prefix string, config map[string]interface{}) ([]string, error) {
	targetDir := s.basePath
	if path, ok := config["path"].(string); ok && path != "" {
		targetDir = path
	}
	
	var files []string
	err := filepath.Walk(targetDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			files = append(files, path)
		}
		return nil
	})
	
	return files, err
}

// copyFile 复制文件
func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()
	
	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()
	
	_, err = io.Copy(destFile, sourceFile)
	if err != nil {
		return err
	}
	
	return destFile.Sync()
}

// S3Storage S3 存储实现（占位符）
type S3Storage struct {
	// TODO: 实现 S3 存储
}

// NewS3Storage 创建 S3 存储
func NewS3Storage() *S3Storage {
	return &S3Storage{}
}

// Upload S3 上传
func (s *S3Storage) Upload(ctx context.Context, localPath string, config map[string]interface{}) (string, error) {
	return "", fmt.Errorf("S3 storage not implemented")
}

// Download S3 下载
func (s *S3Storage) Download(ctx context.Context, remotePath, localPath string, config map[string]interface{}) error {
	return fmt.Errorf("S3 storage not implemented")
}

// Delete S3 删除
func (s *S3Storage) Delete(ctx context.Context, remotePath string, config map[string]interface{}) error {
	return fmt.Errorf("S3 storage not implemented")
}

// Exists S3 检查存在
func (s *S3Storage) Exists(ctx context.Context, remotePath string, config map[string]interface{}) (bool, error) {
	return false, fmt.Errorf("S3 storage not implemented")
}

// List S3 列表
func (s *S3Storage) List(ctx context.Context, prefix string, config map[string]interface{}) ([]string, error) {
	return nil, fmt.Errorf("S3 storage not implemented")
}
