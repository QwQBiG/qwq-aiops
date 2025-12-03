package logger

import (
	"fmt"
	"io"
	"log"
	"os"
	"sync"
	"time"

	"gopkg.in/natefinch/lumberjack.v2"
)

var (
	// WebBuffer 用于 Web 面板显示的内存日志
	WebBuffer []string
	bufferMu  sync.Mutex
	
	// 日志记录器
	infoLogger *log.Logger
)

// 初始化日志系统
func Init(logPath string, debug bool) {
	// 配置日志轮转
	rotator := &lumberjack.Logger{
		Filename:   logPath,
		MaxSize:    10,   // 每个日志文件最大 10MB
		MaxBackups: 5,    // 最多保留 5 个旧文件
		MaxAge:     30,   // 最多保留 30 天
		Compress:   true, // 旧日志压缩保存
	}

	// 多重输出：同时输出到 控制台 + 文件
	multiWriter := io.MultiWriter(os.Stdout, rotator)

	infoLogger = log.New(multiWriter, "", 0) // 时间戳由我们自己格式化
}

// 记录普通日志
func Info(format string, v ...interface{}) {
	msg := fmt.Sprintf(format, v...)
	ts := time.Now().Format("15:04:05")
	logEntry := fmt.Sprintf("[%s] %s", ts, msg)

	// 1. 写入文件和控制台
	if infoLogger != nil {
		infoLogger.Println(logEntry)
	} else {
		fmt.Println(logEntry) // Fallback
	}

	// 2. 写入 Web 内存缓冲 (保留最近 100 条)
	bufferMu.Lock()
	defer bufferMu.Unlock()
	WebBuffer = append(WebBuffer, logEntry)
	if len(WebBuffer) > 100 {
		WebBuffer = WebBuffer[1:]
	}
}

// GetWebLogs 获取 Web 端日志
func GetWebLogs() []string {
	bufferMu.Lock()
	defer bufferMu.Unlock()
	// 返回副本以防并发问题
	logs := make([]string, len(WebBuffer))
	copy(logs, WebBuffer)
	return logs
}