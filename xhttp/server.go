package xhttp

import (
	"context"
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"net/http"
	"strings"
	"time"
)

type Server struct {
	host   *http.Server
	Engine *gin.Engine
}

func NewServer() *Server {
	result := new(Server)
	result.Engine = gin.New()
	result.Engine.Use(result.customLogger(), gin.Recovery())

	return result
}

func (s *Server) Run(port int) error {
	portString := fmt.Sprintf(":%d", port)

	s.host = &http.Server{
		Addr:    portString,
		Handler: s.Engine,
	}

	go func() {
		if err := s.host.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			panic(err)
		}
	}()

	return nil
}

func (s *Server) Stop() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := s.host.Shutdown(ctx); err != nil {
		panic(err)
	}

	<-ctx.Done()
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
