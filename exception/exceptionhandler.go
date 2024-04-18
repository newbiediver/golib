//go:build linux

package exception

import (
	"context"
	"encoding/base64"
	"fmt"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"syscall"
)

type Operator func(callstack string)

type Handler struct {
	userCallstack Operator
	appName       string
	dumpDirectory string
	backendURL    string
}

var (
	exceptionHandler *Handler
)

func StartExceptionHandler(userCallback Operator) *Handler {
	exceptionHandler = new(Handler)
	exceptionHandler.userCallstack = userCallback

	return exceptionHandler
}

func GetExceptionHandler() *Handler {
	return exceptionHandler
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

	var (
		rLimit syscall.Rlimit
	)

	if err := syscall.Getrlimit(syscall.RLIMIT_CORE, &rLimit); err != nil {
		fmt.Println("System can not be called getting rlimit")
		panic(err)
	}
	rLimit.Cur = 0
	rLimit.Max = 0

	if err := syscall.Setrlimit(syscall.RLIMIT_CORE, &rLimit); err != nil {
		panic(err)
	}
}

func (ha *Handler) ExceptionCallbackFunctor() {
	// 덤프는 리눅스만 지원...golang 은 아직 windows mini dump 를 지원하고 있지 않음...
	var (
		coreFile string
	)

	callstack := make([]byte, 4096)
	cnt := runtime.Stack(callstack, false)
	callstackString := string(callstack[:cnt-1])

	hostname, _ := os.Hostname()
	pid := os.Getpid()

	_ = ioutil.WriteFile(fmt.Sprintf("%scallstack.%s.%s.txt", ha.dumpDirectory, hostname, ha.appName), []byte(callstackString), 0644)

	coreFile = fmt.Sprintf("%score.%s.%s", ha.dumpDirectory, hostname, ha.appName)
	cmdCore := exec.Command("gcore", "-o", coreFile, fmt.Sprintf("%d", pid))
	cmdCore.Stdout = os.Stdout

	if err := cmdCore.Run(); err != nil {
		fmt.Println(err)
	}

	if ha.userCallstack != nil {
		ha.userCallstack(callstackString)
	}

	uploadUri := fmt.Sprintf("%s/upload", ha.backendURL)
	contextString := fmt.Sprintf("core.%s", ha.appName)
	imageName := ha.getCurrentDockerImageName()
	base64Callstack := base64.StdEncoding.EncodeToString([]byte(callstackString))

	cmdCrashHandler := exec.Command("sh", "-c", fmt.Sprintf("./crashhub_handler --rm -r %s -d %s -c %s -n %s -i %s -m \"%s\"", uploadUri, ha.dumpDirectory, contextString, ha.appName, imageName, base64Callstack))
	cmdCrashHandler.Stdout = os.Stdout
	if err := cmdCrashHandler.Run(); err != nil {
		fmt.Println(err)
	}
}

func (ha *Handler) getCurrentDockerImageName() string {
	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		return ""
	}

	containers, err := cli.ContainerList(context.Background(), container.ListOptions{All: true})
	if err != nil {
		return ""
	}

	if len(containers) > 0 {
		c := containers[0]
		return c.Image
	}

	return ""
}
