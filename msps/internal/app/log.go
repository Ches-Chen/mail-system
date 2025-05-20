package app

import (
    "context"
    "fmt"
    "io"
    "os"
    "strconv"
    "time"

    "github.com/fatih/color"
    log "github.com/sirupsen/logrus"

    "msps/docs"
)

// CustomTextFormatter 自定义格式化器，继承自 logrus.TextFormatter
type CustomTextFormatter struct {
    log.TextFormatter
    ForceColors   bool
    ColorDebug    *color.Color
    ColorInfo     *color.Color
    ColorWarning  *color.Color
    ColorError    *color.Color
    ColorCritical *color.Color
}

// Format 格式化方法，用于将日志条目格式化为字节数组
func (f *CustomTextFormatter) Format(entry *log.Entry) ([]byte, error) {
    f.FullTimestamp = true
    f.TimestampFormat = time.RFC3339Nano
    if f.ForceColors {
        switch entry.Level {
        case log.DebugLevel:
            f.printColored(entry, f.ColorDebug)
        case log.InfoLevel:
            f.printColored(entry, f.ColorInfo)
        case log.WarnLevel:
            f.printColored(entry, f.ColorWarning)
        case log.ErrorLevel:
            f.printColored(entry, f.ColorError)
        case log.FatalLevel, log.PanicLevel:
            f.printColored(entry, f.ColorCritical)
        default:
            f.printColored(entry, f.ColorInfo)
        }
        return nil, nil
    } else {
        return f.TextFormatter.Format(entry)
    }
}

func (f *CustomTextFormatter) printColored(entry *log.Entry, c *color.Color) {
    levelText := c.Sprintf("[%-5s]", entry.Level.String()) // 格式化日志级别文本
    msg := levelText
    msg = msg + " [" + time.Now().Format(time.DateTime) + "] "
    if entry.HasCaller() {
        msg += "(" + entry.Caller.File + ":" + strconv.Itoa(entry.Caller.Line) + ")" // 添加调用者信息
    }
    msg += " " + c.Sprintf("%s", entry.Message)

    _, _ = fmt.Fprintln(color.Output, msg) // 使用有颜色的方式打印消息到终端
}

// initLog 初始化日志配置
func initLog(ctx context.Context, isDebug bool) {
    if isDebug {
        log.SetLevel(log.DebugLevel)
        log.SetReportCaller(true)
        log.SetOutput(os.Stdout)
    } else {
        log.SetLevel(log.InfoLevel)
        log.SetOutput(io.Discard)
    }

    log.WithContext(ctx)
    log.WithField("app", docs.AppName)
    log.SetFormatter(&CustomTextFormatter{
        ForceColors:   true,
        ColorDebug:    color.New(color.FgGreen),
        ColorInfo:     color.New(color.FgBlue),
        ColorWarning:  color.New(color.FgHiCyan),
        ColorError:    color.New(color.FgRed),
        ColorCritical: color.New(color.BgRed, color.FgWhite),
    })
}
