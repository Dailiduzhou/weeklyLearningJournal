package main

import (
	"os"
	"sync"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

func main() {
	fileWriter := &lumberjack.Logger{
		Filename:   "./log/app.log",
		MaxSize:    1,
		MaxBackups: 5,
		MaxAge:     30,
		Compress:   true,
	}

	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	encoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder
	encoder := zapcore.NewJSONEncoder(encoderConfig)

	core := zapcore.NewCore(
		encoder, zapcore.NewMultiWriteSyncer(zapcore.AddSync(fileWriter), zapcore.AddSync(os.Stdout)), zap.InfoLevel)

	logger := zap.New(core, zap.AddCaller(), zap.AddStacktrace(zap.ErrorLevel))
	defer logger.Sync()

	logger.Info("日志初始化成功", zap.String("status", "ready"))

	var wg sync.WaitGroup
	goroutines := 10
	recordsPerGoroutine := 20

	logger.Info("开始并发写入测试", zap.Int("goroutines", goroutines), zap.Int("recordsPerGoroutine", recordsPerGoroutine))

	for i := range goroutines {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()
			for j := range recordsPerGoroutine {
				logger.Info("并发日志测试",
					zap.Int("goroutineID", goroutineID),
					zap.Int("recordIndex", j),
					zap.String("message", "goroutine正在写入日志"),
					zap.String("nonsense", "dhsjahfkjdnvckcxnzijhawiuoerhiudsahfdsahfsduihfuwbefknxbncvmzxvjisdhfiahwuihjaiHBjksdbckjxzbncjksd H fjkahiwhuadhjsabncbJKHu hjkabfSJHBFhsdDFhjBSKHJFBsyEiuhFUhUHFLJFHJDNxbvshjghjscbvkxzjvhzisuhuiezsfvdxkhfkjdshgsjfhjSDKHfjkszDHfjkzslhdfjhsusehbnbncnxvbhyzsdgfhsdvbnjzddfkjhsLdhsvbgSDGSHJKFhdsFSKVbdxksjhdjgfSD"))
			}
		}(i)
	}

	wg.Wait()
	logger.Info("并发写入测试完成", zap.Int("totalRecords", goroutines*recordsPerGoroutine))

	time.Sleep(100 * time.Millisecond)
	logger.Info("程序结束")
}
