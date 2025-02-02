package scheduler

import (
	"github.com/newbiediver/golib/exception"
	"sync"
	"sync/atomic"
	"time"
)

type procType int
type Priority int

const (
	PriorityVerySlow Priority = 2000
	PrioritySlow     Priority = 1000
	PriorityNormal   Priority = 500
	PriorityFast     Priority = 200
	PriorityVeryFast Priority = 30
	PriorityRealTime Priority = 1
)

const (
	intervalType procType = 0 + iota
	everyDayType
)

const (
	milliSecondToNanoSecond int64 = 1000000
)

type Object struct {
	key          uint64
	interval     int64
	lastTickTime int64
	nextEvent    time.Time
	objType      procType
	completion   func()
	stopFlag     bool
}

type Handler struct {
	termination  bool
	running      bool
	keyContainer uint64
	waiter       sync.WaitGroup
	lock         *sync.Mutex
	newObj       []*Object
	activeObj    map[uint64]*Object
}

var (
	mainHandler  Handler
	keptHandlers map[string]*Handler
)

func GetHandler() *Handler {
	return &mainHandler
}

func GetKeptHandler(name string) *Handler {
	return keptHandlers[name]
}

func KeepHandler(name string, handler *Handler) {
	if keptHandlers == nil {
		keptHandlers = make(map[string]*Handler)
	}

	keptHandlers[name] = handler
}

func StopKeptHandlers() {
	if keptHandlers != nil {
		for _, ha := range keptHandlers {
			ha.Stop()
		}
	}
}

func CreateObjectByInterval(milliSecondInterval int64, completion func()) *Object {
	toNanoSecondInterval := milliSecondInterval * milliSecondToNanoSecond
	obj := new(Object)
	obj.interval = toNanoSecondInterval
	obj.lastTickTime = time.Now().In(time.UTC).UnixNano() + toNanoSecondInterval
	obj.completion = completion
	obj.objType = intervalType
	return obj
}

func CreateObjectByEveryDay(hour int, minute int, second int, completion func()) *Object {
	tomorrow := time.Now().In(time.UTC).Add(24 * time.Hour)
	obj := new(Object)
	obj.nextEvent = time.Date(tomorrow.Year(), tomorrow.Month(), tomorrow.Day(), hour, minute, second, 0, time.UTC)
	obj.objType = everyDayType
	obj.completion = completion
	return obj
}

func (s *Handler) Run(priority Priority) {
	if s.running {
		return
	}

	s.lock = new(sync.Mutex)
	s.waiter.Add(1)
	s.running = true

	go s.procObjects(priority)
}

func (s *Handler) Stop() {
	if !s.running {
		return
	}
	if s.termination {
		return
	}

	s.termination = true
	s.waiter.Wait()
}

func (s *Handler) NewObject(obj *Object) {
	defer s.lock.Unlock()

	s.lock.Lock()
	s.newObj = append(s.newObj, obj)
}

func (s *Handler) DeleteObject(obj *Object) {
	obj.stopFlag = true
}

func (s *Handler) activateObject() {
	if s.newObj == nil {
		return
	}

	defer s.lock.Unlock()
	s.lock.Lock()

	if s.activeObj == nil {
		s.activeObj = make(map[uint64]*Object)
	}

	if s.newObj != nil {
		for _, obj := range s.newObj {
			newKey := atomic.AddUint64(&s.keyContainer, 1)
			obj.key = newKey
			s.activeObj[newKey] = obj
		}
		s.newObj = nil
	}
}

func (s *Handler) procObjects(p Priority) {
	defer func() {
		if rcv := recover(); rcv != nil {
			if ex := exception.GetExceptionHandler(); ex != nil {
				ex.ExceptionCallbackFunctor()
			}
		}
		s.waiter.Done()
	}()

	var stopObjects []*Object

	for !s.termination {
		s.activateObject()
		now := time.Now().In(time.UTC).UnixNano()
		for _, obj := range s.activeObj {
			if obj.stopFlag {
				stopObjects = append(stopObjects, obj)
				continue
			}
			if obj.objType == intervalType {
				if now >= obj.lastTickTime {
					obj.completion()
					obj.lastTickTime = now + obj.interval
				}
			} else {
				if now >= obj.nextEvent.UnixNano() {
					tomorrow := time.Now().In(time.UTC).Add(24 * time.Hour)
					obj.completion()
					obj.nextEvent = time.Date(tomorrow.Year(), tomorrow.Month(), tomorrow.Day(), obj.nextEvent.Hour(), obj.nextEvent.Minute(), obj.nextEvent.Second(), 0, time.UTC)
				}
			}
		}

		if stopObjects != nil {
			for _, obj := range stopObjects {
				s.removeObject(obj)
			}
		}

		time.Sleep(time.Millisecond * time.Duration(p))
	}
}

func (s *Handler) removeObject(obj *Object) {
	delete(s.activeObj, obj.key)
}
