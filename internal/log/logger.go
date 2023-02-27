package log

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"
)

type LogEvent interface {
	WithField(field string, value any) LogEvent
	WithFields(fields map[string]any) LogEvent
	WithError(err error) LogEvent
	Info(msg string)
	Error(msg string)
	Fatal(msg string)
	AddToContext(ctx context.Context)
}

func New() LogEvent {
	return &event{}
}

type event struct {
	timestamp time.Time
	msg       string
	level     string
	fields    map[string]any
}

func (e *event) Info(msg string) {
	e.timestamp = time.Now().In(time.UTC)
	e.msg = msg
	e.level = "info"
	send(e)
}

func (e *event) Error(msg string) {
	e.timestamp = time.Now().In(time.UTC)
	e.msg = msg
	e.level = "error"
	send(e)
}

func (e *event) Fatal(msg string) {
	e.timestamp = time.Now().In(time.UTC)
	e.msg = msg
	e.level = "error"
	send(e)
	ctx, _ := context.WithTimeout(context.Background(), time.Second)
	Drain(ctx)
	os.Exit(1)
}

func (e *event) AddToContext(ctx context.Context) {
	lf, ok := ctx.Value(contextKey{}).(map[string]any)
	if !ok {
		e.Error("couldn't add log fields to context, make sure context is created using logging middleware")
		return
	}
	if e.fields != nil {
		for k, v := range e.fields {
			lf[k] = v
		}
	}
}

func (e *event) WithField(field string, value any) LogEvent {
	if e.fields == nil {
		e.fields = make(map[string]any)
	}
	e.fields[field] = value
	return e
}

func (e *event) WithFields(fields map[string]any) LogEvent {
	if e.fields == nil {
		e.fields = make(map[string]any)
	}
	for k, v := range fields {
		e.fields[k] = v
	}
	return e
}

func (e *event) WithError(err error) LogEvent {
	if err == nil {
		return e
	}
	return e.WithField("error_message", err.Error())
}

var _queue chan event
var _ctx context.Context

func init() {
	_queue = make(chan event, 512)
	var cancel context.CancelFunc
	_ctx, cancel = context.WithCancel(context.Background())
	go func() {
		defer cancel()
		for e := range _queue {
			if e.fields == nil {
				e.fields = make(map[string]any)
			}
			e.fields["timestamp"] = e.timestamp.Format("2006-01-02T15:04:05.99Z07:00")
			e.fields["msg"] = e.msg
			e.fields["level"] = e.level
			b, _ := json.Marshal(e.fields)
			fmt.Println(string(b))
		}
	}()
}

func send(e *event) {
	select {
	case _queue <- *e:
	default:
	}
}

var _mu sync.Mutex
var _closed bool

func Drain(ctx context.Context) {
	_mu.Lock()
	if !_closed {
		close(_queue)
		_closed = true
	}
	drainedCtx := _ctx
	_mu.Unlock()
	select {
	case <-drainedCtx.Done():
	case <-ctx.Done():
	}
}
