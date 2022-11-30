package logger

import "fmt"

type LogLevel uint64

const (
	None LogLevel = iota + 1
	Critical
	Info
	Debug
	Error
)

func (l LogLevel) String() string {
	return [...]string{"None", "Critical", "Info", "Debug", "Error"}[l-1]
}

func (l LogLevel) EnumIndex() int {
	return int(l)
}

type BaseLogger struct {
	LogLevel LogLevel
}

func NewBaseLogger(lv LogLevel) *BaseLogger {
	return &BaseLogger{
		LogLevel: lv,
	}
}

func (bl *BaseLogger) PrintCritical(str string) {
	if bl.LogLevel >= Critical {
		fmt.Print(str)
	}
}

func (bl *BaseLogger) PrintError(str string) {
	if bl.LogLevel >= Error {
		fmt.Print(str)
	}
}

func (bl *BaseLogger) PrintInfo(str string) {
	if bl.LogLevel >= Info {
		fmt.Print(str)
	}
}

func (bl *BaseLogger) PrintDebug(str string) {
	if bl.LogLevel >= Debug {
		fmt.Print(str)
	}
}
