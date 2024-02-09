package eventlog

import (
	"bytes"
	"context"
	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/hashicorp/go-multierror"
	"github.com/jsiebens/ionscale/internal/config"
	"github.com/jsiebens/ionscale/internal/util"
	"go.uber.org/zap"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"
)

const (
	stdout  = "/dev/stdout"
	stderr  = "/dev/stderr"
	devnull = "/dev/null"
)

type Events []cloudevents.Event

func (e *Events) Add(event cloudevents.Event) {
	x := append(*e, event)
	*e = x
}

type Eventer interface {
	Send(ctx context.Context, events ...cloudevents.Event) error
}

type eventer struct {
	source string
	sinks  []sink
}

func (e *eventer) Send(ctx context.Context, events ...cloudevents.Event) error {
	groupID := util.NextIDString()
	now := time.Now()

	for _, event := range events {
		event.SetSource(e.source)
		event.SetID(util.NextIDString())
		event.SetTime(now)
		event.SetExtension("eventGroupID", groupID)
	}

	var r *multierror.Error
	for _, s := range e.sinks {
		r = multierror.Append(r, s.process(ctx, events...))
	}

	return r.ErrorOrNil()
}

type sink interface {
	process(context.Context, ...cloudevents.Event) error
}

var (
	_globalMu sync.RWMutex
	_globalE  Eventer = &eventer{}
)

func Configure(c *config.Config) error {
	var sinks []sink

	if c.Events.Log.Enabled {
		sinks = append(sinks, &zapSink{logger: zap.L().Named("events").WithOptions(zap.AddCallerSkip(3))})
	}

	if c.Events.File.Enabled {
		switch c.Events.File.Path {
		case devnull:
			// ignore
		case stderr:
			sinks = append(sinks, &writerSink{w: os.Stderr})
		case stdout:
			sinks = append(sinks, &writerSink{w: os.Stdout})
		default:
			abs, err := filepath.Abs(c.Events.File.Path)
			if err != nil {
				return err
			}

			sinks = append(sinks, &fileSink{
				path:        abs,
				fileName:    c.Events.File.FileName,
				maxBytes:    c.Events.File.MaxBytes,
				maxDuration: c.Events.File.MaxDuration,
				maxFiles:    c.Events.File.MaxFiles,
			})
		}
	}

	_globalMu.Lock()
	defer _globalMu.Unlock()
	_globalE = &eventer{
		source: c.ServerUrl,
		sinks:  sinks,
	}

	return nil
}

func Send(ctx context.Context, events ...cloudevents.Event) {
	_globalMu.RLock()
	l := _globalE
	_globalMu.RUnlock()

	if err := l.Send(ctx, events...); err != nil {
		zap.L().Error("error while processing event", zap.Error(err))
	}
}

func writeJSONLine(w io.Writer, events ...cloudevents.Event) (int, error) {
	var payload bytes.Buffer

	for _, event := range events {
		eventJson, err := event.MarshalJSON()
		if err != nil {
			return 0, err
		}

		payload.Write(eventJson)
		payload.Write([]byte("\n"))
	}

	return w.Write(payload.Bytes())
}
