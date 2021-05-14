package logger

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

var logger *zap.SugaredLogger

func init() {

	lvl := os.Getenv("LOG_LVL")
	if len(lvl) <= 0 {
		lvl = "info"
	}

	InitLog(setLogFile(info), setLogFile(err), lvl, false)
}

func InitLog(logPath, errPath string, level string, enable bool) {
	config := zapcore.EncoderConfig{
		MessageKey:  "msg",
		LevelKey:    "level",
		EncodeLevel: zapcore.CapitalLevelEncoder, //将级别转换成大写
		TimeKey:     "ts",
		EncodeTime: func(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
			enc.AppendString(t.Format("2006-01-02 15:04:05"))
		},
		CallerKey:    "file",
		EncodeCaller: zapcore.ShortCallerEncoder,
		EncodeDuration: func(d time.Duration, enc zapcore.PrimitiveArrayEncoder) {
			enc.AppendInt64(int64(d) / 1000000)
		},
	}
	encoder := zapcore.NewConsoleEncoder(config)
	// 设置级别
	logLevel := zap.DebugLevel
	switch level {
	case "debug":
		logLevel = zap.DebugLevel
	case "info":
		logLevel = zap.InfoLevel
	case "warn":
		logLevel = zap.WarnLevel
	case "error":
		logLevel = zap.ErrorLevel
	case "panic":
		logLevel = zap.PanicLevel
	case "fatal":
		logLevel = zap.FatalLevel
	default:
		logLevel = zap.InfoLevel
	}
	// 实现两个判断日志等级的interface  可以自定义级别展示
	infoLevel := zap.LevelEnablerFunc(func(lvl zapcore.Level) bool {
		return lvl < zapcore.WarnLevel && lvl >= logLevel
	})

	warnLevel := zap.LevelEnablerFunc(func(lvl zapcore.Level) bool {
		return lvl >= zapcore.WarnLevel && lvl >= logLevel
	})

	var atomicLevel = zap.NewAtomicLevelAt(logLevel)

	// 最后创建具体的Logger
	core := zapcore.NewTee(
		// 将info及以下写入logPath,  warn及以上写入errPath
		zapcore.NewCore(encoder, getLogWriter(logPath), infoLevel),
		zapcore.NewCore(encoder, getLogWriter(errPath), warnLevel),
		//日志都会在console中展示
		zapcore.NewCore(zapcore.NewConsoleEncoder(config),
			zapcore.NewMultiWriteSyncer(zapcore.AddSync(os.Stdout)), atomicLevel),
	)
	log := zap.New(core, zap.AddCaller(), zap.AddStacktrace(zap.WarnLevel)) // 需要传入 zap.AddCaller() 才会显示打日志点的文件名和行数, 有点小坑
	logger = log.Sugar()
	logger.Sync()

	if enable {
		http.HandleFunc("/app/level", atomicLevel.ServeHTTP) //动态修改日志api
		go func() {
			if err := http.ListenAndServe(":9090", nil); err != nil {
				panic(err)
			}
		}()
	}
}

func getLogWriter(filename string) zapcore.WriteSyncer {
	lumberJackLogger := &lumberjack.Logger{
		Filename:   filename, // 日志文件位置
		MaxSize:    10,         // 日志文件最大大小(MB)
		MaxBackups: 5,          // 保留旧文件最大数量
		MaxAge:     30,         // 保留旧文件最长天数
		Compress:   false,      // 是否压缩旧文件
	}
	return zapcore.AddSync(lumberJackLogger)
}

type logFileType int

const (
	err logFileType = iota +1
	info
)


func setLogFile(T logFileType) string {
	if T == err {
		return getCurrentDirectory() + "/" + getAppName() + "_err.log"
	}
	return getCurrentDirectory() + "/" + getAppName() + "_.log"
}

func getCurrentDirectory() string {
	pwd, _ := os.Getwd()
	return pwd
}

func getAppName() string {
	path := os.Args[0]
	_, fname := filepath.Split(path)
	return fname
}

func Debug(format string, v...interface{}) {
	logger.Debugf(format, v...)
}

func Info(format string, v ...interface{}) {
	logger.Infof(format, v...)
}

func Warn(format string, v ...interface{}) {
	logger.Warnf(format, v...)
}

func Error(format string, v ...interface{}) {
	logger.Errorf(format, v...)
}

func Panic(format string, v ...interface{}) {
	logger.Panicf(format, v...)
}
