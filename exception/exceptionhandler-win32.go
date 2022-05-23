//go:build windows

package exception

import (
	"os"
	"path/filepath"
)

type Operator func(callstack string)

type Handler struct {
	userCallstack Operator
	appName       string
	dumpDirectory string
	backendURL    string
}

func NewExceptionHandler(userCallback Operator) *Handler {
	r := new(Handler)
	r.userCallstack = userCallback

	return r
}

func (ha *Handler) RunWithCrashHub(appName, dumpDirectory, backend string) {
	if dumpDirectory[len(dumpDirectory)-1] != '/' && dumpDirectory[len(dumpDirectory)-1] != '\\' {
		dumpDirectory += "/"
	}

	if _, err := os.Stat(dumpDirectory); os.IsNotExist(err) {
		_ = os.Mkdir(dumpDirectory, 0755)
	}

	ha.appName = appName
	ha.dumpDirectory = filepath.FromSlash(dumpDirectory)
	ha.backendURL = backend
}

func (ha *Handler) ExceptionCallbackFunctor() {
	// 덤프는 리눅스만 지원...golang 은 아직 windows mini dump 를 지원하고 있지 않음...

}
