package xlog

import (
	"fmt"
	"github.com/newbiediver/golib/scheduler"
	"os"
	"sync"
	"time"
)

type LogLevel int

const (
	information LogLevel = iota
	warning
	error
	fatal
)

type logObject struct {
	timeString 		string
	bodyString		string
	level			LogLevel
}

type logger struct {
	appName			string
	logs			[]logObject
	loc 			*time.Location
	sc				*scheduler.Handler
	lock 			*sync.Mutex
}

var (
	curLogger	logger
)

func RunLogger(runningScheduler *scheduler.Handler, appName string, loc *time.Location) {
	if _, err := os.Stat("./Log"); os.IsNotExist(err) {
		_ = os.Mkdir("./Log", 0755)
	}

	curLogger.appName = appName
	curLogger.loc = loc
	curLogger.lock = new(sync.Mutex)

	obj := scheduler.CreateObjectByInterval(60000, curLogger.procSchedule)
	runningScheduler.NewObject(obj)
}

func StopLogger() {
	curLogger.procSchedule()
}

func printf(lv LogLevel, format string, a ...interface{}) {
	str := fmt.Sprintf(format, a...)
	now := time.Now().In(curLogger.loc)
	timeString := fmt.Sprintf("%04d.%02d.%02d %02d:%02d:%02d", now.Year(), now.Month(), now.Day(), now.Hour(), now.Minute(), now.Second())

	obj := logObject{
		timeString: timeString,
		bodyString: str,
		level: lv,
	}

	curLogger.lock.Lock()
	curLogger.logs = append(curLogger.logs, obj)
	curLogger.lock.Unlock()

	levelString := []string{
		"INFO ",
		"WARN ",
		"ERROR",
		"FATAL",
	}

	fmt.Printf("[%s] [%s] %s\n", timeString, levelString[lv], str)
}

func Info(format string, a ...interface{}) {
	printf(information, format, a...)
}

func Warn(format string, a ...interface{}) {
	printf(warning, format, a...)
}

func Error(format string, a ...interface{}) {
	printf(error, format, a...)
}

func Fatal(format string, a ...interface{}) {
	printf(fatal, format, a...)
}

func timeToString(tm time.Time) string {
	return fmt.Sprintf("%04d.%02d.%02d %02d:%02d:%02d", tm.Year(), tm.Month(), tm.Day(), tm.Hour(), tm.Minute(), tm.Second())
}

func (l *logger) procSchedule() {
	defer func() {
		if r := recover(); r != nil {
			now := time.Now().In(l.loc)
			fmt.Printf("[%s] [FATAL] %s\n", timeToString(now), r)
		}
	}()

	if l.logs == nil {
		return
	}

	levelString := []string{
		"INFO ",
		"WARN ",
		"ERROR",
		"FATAL",
	}

	now := time.Now().In(l.loc)
	filePath := fmt.Sprintf("./Log/%s_%04d-%02d-%02d.log", l.appName, now.Year(), now.Month(), now.Day())

	file, err := os.OpenFile(filePath, os.O_APPEND | os.O_WRONLY | os.O_CREATE, 0644)
	if err != nil {
		panic("Could not open log file!")
	}

	defer file.Close()

	l.lock.Lock()

	for _, logItem := range l.logs {
		str := fmt.Sprintf("[%s] [%s] %s", logItem.timeString, levelString[logItem.level], logItem.bodyString)
		fmt.Fprintln(file, str)
	}

	l.logs = nil
	l.lock.Unlock()
}