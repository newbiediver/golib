package xredis

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/newbiediver/golib/container"
	"github.com/redis/go-redis/v9"
)

type endpoint struct {
	address string
	pwd     string
	port    int
	index   int
}

type Handler struct {
	redisHandler *redis.Client
}

var (
	ep              endpoint
	managedHandlers container.SafeQueue[*Handler]
)

func CreateHandlers(address, pwd string, port, index, io int) error {
	ep.address = address
	ep.pwd = pwd
	ep.port = port
	ep.index = index

	for i := 0; i < io; i++ {
		handler, err := createHandler()
		if err != nil {
			return err
		}

		managedHandlers.Push(handler)
	}

	return nil
}

func createHandler() (*Handler, error) {
	ctx := context.Background()

	handler := new(Handler)
	handler.redisHandler = redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", ep.address, ep.port),
		Password: ep.pwd,
		DB:       ep.index,
	})

	if err := handler.redisHandler.Ping(ctx).Err(); err != nil {
		return nil, err
	}

	return handler, nil
}

func AllocateHandler() *Handler {
	if handler, ok := managedHandlers.Pop(); ok {
		return handler
	}

	if managedHandlers.Len() == 0 {
		handler, err := createHandler()
		if err != nil {
			panic(err)
		}

		return handler
	}

	return nil
}

func ReleaseHandler(h *Handler) {
	managedHandlers.Push(h)
}

func FlushHandlers() {
	for managedHandlers.Len() > 0 {
		if handler, ok := managedHandlers.Pop(); ok {
			handler.Close()
		}
	}
}

func (h *Handler) Close() {
	_ = h.redisHandler.Close()
}

func (h *Handler) HashSet(key, field string, value any) error {
	ctx := context.Background()
	b, err := json.Marshal(value)
	if err != nil {
		return err
	}
	if err := h.redisHandler.HSet(ctx, key, field, b).Err(); err != nil {
		return err
	}

	return nil
}

func (h *Handler) HashGet(key, field string) (any, error) {
	ctx := context.Background()
	value, err := h.redisHandler.HGet(ctx, key, field).Result()
	if err != nil {
		return nil, err
	}

	return value, nil
}

func (h *Handler) Exists(key string) bool {
	ctx := context.Background()
	exists, err := h.redisHandler.Exists(ctx, key).Result()
	if err != nil {
		return false
	}

	return exists == 1
}
