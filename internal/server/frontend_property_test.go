// Package server 前端资源属性测试
// 使用 gopter 库进行属性基础测试，验证前端资源管理的正确性
package server

import (
	"fmt"
	"io/fs"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
)

// ============================================
// 测试辅助函数 - 模拟文件系统创建
// ============================================

// createMockFS 创建模拟文件系统
// 参数：hasIndex - 是否包含 index.html
//
//	hasJS - 是否包含 JS 文件
//	hasCSS - 是否包含 CSS 文件
func createMockFS(hasIndex, hasJS, hasCSS bool) fs.FS {
	files := make(fstest.MapFS)

	if hasIndex {
		files["dist/index.html"] = &fstest.MapFile{
			Data: []byte("<!DOCTYPE html><html><head></head><body><script src=\"/assets/main.js\"></script></body></html>"),
		}
	}

	if hasJS {
		files["dist/assets/main.js"] = &fstest.MapFile{
			Data: []byte("console.log('hello');"),
		}
	}

	if hasCSS {
		files["dist/assets/style.css"] = &fstest.MapFile{
			Data: []byte("body { margin: 0; }"),
		}
	}

	return files
}

// createMockFSWithExtras 创建包含额外文件的模拟文件系统
func createMockFSWithExtras(hasIndex, hasJS, hasCSS bool, extraFiles int) fs.FS {
	files := make(fstest.MapFS)

	if hasIndex {
		files["dist/index.html"] = &fstest.MapFile{
			Data: []byte("<!DOCTYPE html><html><head></head><body><script src=\"/assets/main.js\"></script></body></html>"),
		}
	}

	if hasJS {
		files["dist/assets/main.js"] = &fstest.MapFile{
			Data: []byte("console.log('hello');"),
		}
	}

	if hasCSS {
		files["dist/assets/style.css"] = &fstest.MapFile{
			Data: []byte("body { margin: 0; }"),
		}
	}

	// 添加额外文件
	for i := 0; i < extraFiles; i++ {
		files[fmt.Sprintf("dist/assets/extra%d.js", i)] = &fstest.MapFile{
			Data: []byte(fmt.Sprintf("// extra file %d", i)),
		}
	}

	return files
}

// createMockFSWithCounts 创建指定数量 JS/CSS 文件的模拟文件系统
func createMockFSWithCounts(hasIndex bool, jsCount, cssCount int) fs.FS {
	files := make(fstest.MapFS)

	if hasIndex {
		files["dist/index.html"] = &fstest.MapFile{
			Data: []byte("<!DOCTYPE html><html><head></head><body></body></html>"),
		}
	}

	// 创建指定数量的 JS 文件
	for i := 0; i < jsCount; i++ {
		files[fmt.Sprintf("dist/assets/script%d.js", i)] = &fstest.MapFile{
			Data: []byte(fmt.Sprintf("// script %d", i)),
		}
	}

	// 创建指定数量的 CSS 文件
	for i := 0; i < cssCount; i++ {
		files[fmt.Sprintf("dist/assets/style%d.css", i)] = &fstest.MapFile{
			Data: []byte(fmt.Sprintf("/* style %d */", i)),
		}
	}

	return files
}

// createMockFSWithContent 创建包含指定内容文件的模拟文件系统
func createMockFSWithContent(filename, content string) fs.FS {
	files := make(fstest.MapFS)
	files["dist/"+filename] = &fstest.MapFile{
		Data: []byte(content),
	}
	return files
}

// ============================================
// 属性测试 1: 前端资源完整性验证
// ============================================

// **Feature: deployment-ai-config-fix, Property 1: 前端资源完整性验证**
// **Validates: Requirements 1.1, 1.3**
//
// 验证内容：
// 1. 缺失必需文件时验证应该失败
// 2. 所有必需文件存在时验证应该通过
// 3. 损坏的文件应该被检测到
// 4. 验证结果包含有用的错误信息和修复建议
func TestProperty1_FrontendResourceIntegrityValidation(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100
	properties := gopter.NewProperties(parameters)

	// 属性 1: 缺失 index.html 时验证应该失败
	properties.Property("缺失index.html时验证失败", prop.ForAll(
		func(hasAssets bool) bool {
			mockFS := createMockFS(false, hasAssets, true)
			fm := NewFrontendManagerWithFS(mockFS, "dist")

			result := fm.ValidateResources()

			// 验证应该失败，错误信息应该提到 index.html
			if result.Valid {
				return false
			}

			for _, err := range result.Errors {
				if strings.Contains(err, "index.html") {
					return true
				}
			}
			return false
		},
		gen.Bool(),
	))

	// 属性 2: 所有必需文件存在时验证应该通过
	properties.Property("所有必需文件存在时验证通过", prop.ForAll(
		func(extraFiles int) bool {
			mockFS := createMockFSWithExtras(true, true, true, extraFiles)
			fm := NewFrontendManagerWithFS(mockFS, "dist")

			result := fm.ValidateResources()
			return result.Valid && len(result.Errors) == 0
		},
		gen.IntRange(0, 10),
	))

	// 属性 3: 缺失 JS 文件时验证应该失败
	properties.Property("缺失JS文件时验证失败", prop.ForAll(
		func(hasCss bool) bool {
			mockFS := createMockFS(true, false, hasCss)
			fm := NewFrontendManagerWithFS(mockFS, "dist")

			result := fm.ValidateResources()

			if result.Valid {
				return false
			}

			for _, err := range result.Errors {
				if strings.Contains(err, "JavaScript") {
					return true
				}
			}
			return false
		},
		gen.Bool(),
	))

	// 属性 4: 验证失败时应该包含修复建议
	properties.Property("验证失败时包含修复建议", prop.ForAll(
		func(hasIndex, hasJS bool) bool {
			mockFS := createMockFS(hasIndex, hasJS, true)
			fm := NewFrontendManagerWithFS(mockFS, "dist")

			result := fm.ValidateResources()

			// 如果验证失败，应该有修复建议
			if !result.Valid {
				return len(result.Suggestions) > 0
			}
			return true
		},
		gen.Bool(),
		gen.Bool(),
	))

	// 属性 5: 资源统计应该准确
	properties.Property("资源统计准确", prop.ForAll(
		func(jsCount, cssCount int) bool {
			if jsCount < 0 || jsCount > 20 || cssCount < 0 || cssCount > 20 {
				return true
			}

			mockFS := createMockFSWithCounts(true, jsCount, cssCount)
			fm := NewFrontendManagerWithFS(mockFS, "dist")

			stats := fm.GetResourceStats()
			return stats.JSFiles == jsCount && stats.CSSFiles == cssCount
		},
		gen.IntRange(0, 20),
		gen.IntRange(0, 20),
	))

	// 属性 6: 文件哈希一致性
	properties.Property("文件哈希一致性", prop.ForAll(
		func(content string) bool {
			if len(content) == 0 {
				return true
			}

			mockFS := createMockFSWithContent("test.js", content)
			fm := NewFrontendManagerWithFS(mockFS, "dist")

			hash1, err1 := fm.GetFileHash("test.js")
			hash2, err2 := fm.GetFileHash("test.js")

			if err1 != nil || err2 != nil {
				return false
			}
			return hash1 == hash2
		},
		gen.AnyString().SuchThat(func(s string) bool { return len(s) > 0 && len(s) < 1000 }),
	))

	// 属性 7: 空文件应该被检测为损坏
	properties.Property("空文件被检测为损坏", prop.ForAll(
		func(filename string) bool {
			if filename == "" || strings.Contains(filename, "/") {
				return true
			}

			mockFS := createMockFSWithContent(filename, "")
			fm := NewFrontendManagerWithFS(mockFS, "dist")

			err := fm.ValidateFileIntegrity(filename)
			return err != nil && strings.Contains(err.Error(), "内容为空")
		},
		gen.RegexMatch("[a-z]+\\.[a-z]+"),
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// ============================================
// 属性测试 2: 前端资源嵌入一致性
// ============================================

// **Feature: deployment-ai-config-fix, Property 2: 前端资源嵌入一致性**
// **Validates: Requirements 1.2**
//
// 验证内容：
// 1. 嵌入的文件内容与原始内容一致
// 2. 文件哈希值保持不变
// 3. Content-Type 正确设置
func TestProperty2_FrontendResourceEmbedConsistency(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100
	properties := gopter.NewProperties(parameters)

	// 属性 1: 嵌入内容一致性
	properties.Property("嵌入内容一致性", prop.ForAll(
		func(content string) bool {
			if len(content) == 0 || len(content) > 10000 {
				return true
			}

			mockFS := createMockFSWithContent("test.txt", content)
			fm := NewFrontendManagerWithFS(mockFS, "dist")

			readContent, err := fm.ReadFile("test.txt")
			if err != nil {
				return false
			}
			return string(readContent) == content
		},
		gen.AnyString().SuchThat(func(s string) bool { return len(s) > 0 && len(s) < 10000 }),
	))

	// 属性 2: 不同内容产生不同哈希
	properties.Property("不同内容产生不同哈希", prop.ForAll(
		func(content1, content2 string) bool {
			if content1 == content2 || len(content1) == 0 || len(content2) == 0 {
				return true
			}

			mockFS1 := createMockFSWithContent("test.js", content1)
			mockFS2 := createMockFSWithContent("test.js", content2)

			fm1 := NewFrontendManagerWithFS(mockFS1, "dist")
			fm2 := NewFrontendManagerWithFS(mockFS2, "dist")

			hash1, err1 := fm1.GetFileHash("test.js")
			hash2, err2 := fm2.GetFileHash("test.js")

			if err1 != nil || err2 != nil {
				return false
			}
			return hash1 != hash2
		},
		gen.AnyString().SuchThat(func(s string) bool { return len(s) > 0 && len(s) < 1000 }),
		gen.AnyString().SuchThat(func(s string) bool { return len(s) > 0 && len(s) < 1000 }),
	))

	// 属性 3: Content-Type 正确设置
	properties.Property("Content-Type正确设置", prop.ForAll(
		func(ext string) bool {
			expectedTypes := map[string]string{
				".html":  "text/html; charset=utf-8",
				".js":    "application/javascript; charset=utf-8",
				".css":   "text/css; charset=utf-8",
				".json":  "application/json; charset=utf-8",
				".png":   "image/png",
				".jpg":   "image/jpeg",
				".jpeg":  "image/jpeg",
				".svg":   "image/svg+xml",
				".ico":   "image/x-icon",
				".woff":  "font/woff",
				".woff2": "font/woff2",
				".ttf":   "font/ttf",
			}

			expected, ok := expectedTypes[ext]
			if !ok {
				return true
			}

			actual := GetContentType("test" + ext)
			return actual == expected
		},
		gen.OneConstOf(".html", ".js", ".css", ".json", ".png", ".jpg", ".jpeg", ".svg", ".ico", ".woff", ".woff2", ".ttf"),
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}
