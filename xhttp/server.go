package xhttp

import (
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"net/http"
	"strings"
	"time"
)

type HttpHandler struct {
	Method          MethodType
	AsFunctor       func(*gin.Context)
	AsDirectoryPath string
	AsFilePath      string
}

type Server struct {
	engine *gin.Engine
	binder map[string]*HttpHandler
}

func NewServer() *Server {
	result := new(Server)
	result.binder = make(map[string]*HttpHandler)
	result.engine = gin.New()
	result.engine.Use(result.customLogger(), gin.Recovery())

	return result
}

func (s *Server) Bind(path string, handler *HttpHandler) {
	s.binder[path] = handler
}

func (s *Server) Run(port int) error {
	portString := fmt.Sprintf(":%d", port)
	if len(s.binder) == 0 {
		return errors.New("no path bound")
	}

	for path, handler := range s.binder {
		switch handler.Method {
		case GET:
			if handler.AsDirectoryPath != "" {
				s.engine.Static(path, handler.AsDirectoryPath)
			} else if handler.AsFilePath != "" {
				s.engine.StaticFile(path, handler.AsFilePath)
			} else {
				s.engine.GET(path, handler.AsFunctor)
			}
			break
		case POST:
			s.engine.POST(path, handler.AsFunctor)
			break
		case DELETE:
			s.engine.DELETE(path, handler.AsFunctor)
			break
		}
	}

	go func() {
		if err := http.ListenAndServe(portString, s.engine); err != nil && !errors.Is(err, http.ErrServerClosed) {
			panic(err)
		}
	}()

	return nil
}

func (s *Server) customLogger() gin.HandlerFunc {
	out := gin.DefaultWriter

	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		end := time.Now()

		latency := end.Sub(start)
		proxyHeader := c.GetHeader("X-Forwarded-For")

		var clientAddress string

		if proxyHeader != "" {
			ips := strings.Split(proxyHeader, ",")
			if len(ips) > 0 {
				clientAddress = ips[0]
			} else {
				clientAddress = proxyHeader
			}
		} else {
			clientAddress = c.ClientIP()
		}

		timeString := time.Now().In(time.UTC).Format("2006/01/02 - 15:04:05")
		format := fmt.Sprintf("[System] %v | %3d | %13v | %15s | %-7s %#v\n",
			timeString,
			c.Writer.Status(),
			latency,
			clientAddress,
			c.Request.Method,
			c.Request.RequestURI)

		_, _ = fmt.Fprintf(out, format)
	}
}
