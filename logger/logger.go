package logger

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"
)

type LoggerInterface interface {
	r(string, ...any)
}

type Logger struct {
	Level       int
	FileDst     string
	AddLogLevel bool
	AddDateTime bool
}

const (
	DEBUG   = 0
	INFO    = 1
	WARNING = 2
	ERROR   = 3
)

func baseLog(l *Logger, str string, args ...any) {
	if !strings.HasSuffix(str, "\n") {
		str += "\n"
	}

	if l.AddDateTime {
		logDateTime := time.Now().UTC().Format(time.RFC3339)
		str = fmt.Sprintf("[%s] %s", logDateTime, str)
	}

	if l.FileDst != "" {
		fileDesc, err := os.OpenFile(l.FileDst, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0644)
		if err != nil {
			panic(err)
		}
		src := fmt.Sprintf(str, args...)
		ret, err := fileDesc.Write([]byte(src))
		if err != nil {
			panic(err)
		}
		if len(src) != ret {
			panic(src)
		}
	}

	fmt.Printf(str, args...)
}

func baseLogBytes(l *Logger, src []byte) {
	if l.FileDst != "" {
		fileDesc, err := os.OpenFile(l.FileDst, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0644)
		if err != nil {
			panic(err)
		}
		ret, err := fileDesc.Write(src)
		if err != nil {
			panic(err)
		}
		if len(src) != ret {
			panic(src)
		}
	}

	fmt.Print(string(src))
}

func (l *Logger) baseM(srcMap map[string]any) {
	if l.Level > INFO {
		return
	}

	if l.AddDateTime {
		srcMap["LogDateTime"] = time.Now().UTC().Format(time.RFC3339)
	}
	jsonData, err := json.Marshal(srcMap)
	if err != nil {
		baseLog(l, "Error marshaling JSON: %v", err)
		return
	}
	jsonData = append(jsonData, byte('\n'))
	baseLogBytes(l, jsonData)
}

func (l *Logger) InfoM(srcMap map[string]any) {
	if l.Level > INFO {
		return
	}
	if l.AddLogLevel {
		srcMap["LogLevel"] = "INFO"
	}
	l.baseM(srcMap)
}

func (l *Logger) InfoF(str string, args ...any) {
	if l.Level > INFO {
		return
	}
	baseLog(l, str, args...)

}

func (l *Logger) DebugF(str string, args ...any) {
	if l.Level > DEBUG {
		return
	}
	baseLog(l, str, args...)
}

func (l *Logger) WarningF(str string, args ...any) {
	if l.Level > WARNING {
		return
	}
	baseLog(l, str, args...)
}

func (l *Logger) ErrorF(str string, args ...any) {
	if l.Level > ERROR {
		return
	}
	baseLog(l, str, args...)
}
