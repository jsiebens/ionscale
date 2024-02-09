package eventlog

import (
	"context"
	"fmt"
	cloudevents "github.com/cloudevents/sdk-go/v2"
	"go.uber.org/zap"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	fileMode = os.FileMode(0600)
	dirMode  = os.FileMode(0700)
)

type zapSink struct {
	logger *zap.Logger
}

func (z *zapSink) process(_ context.Context, events ...cloudevents.Event) error {
	for _, event := range events {
		z.logger.Info(event.Type(), zap.String("data", string(event.Data())))
	}
	return nil
}

type writerSink struct {
	w io.Writer
}

func (w *writerSink) process(_ context.Context, events ...cloudevents.Event) error {
	_, err := writeJSONLine(w.w, events...)
	return err
}

type fileSink struct {
	path        string
	fileName    string
	maxBytes    int
	maxDuration time.Duration
	maxFiles    int

	lastCreated  time.Time
	bytesWritten int64

	f *os.File
	l sync.Mutex
}

func (fs *fileSink) process(_ context.Context, events ...cloudevents.Event) error {
	fs.l.Lock()
	defer fs.l.Unlock()

	if fs.f == nil {
		err := fs.open()
		if err != nil {
			return err
		}
	}

	if err := fs.rotate(); err != nil {
		return err
	}

	if n, err := writeJSONLine(fs.f, events...); err == nil {
		fs.bytesWritten += int64(n)
		return nil
	}

	if err := fs.reopen(); err != nil {
		return err
	}

	_, err := writeJSONLine(fs.f, events...)

	return err
}

func (fs *fileSink) reopen() error {
	if fs.f != nil {
		_, err := os.Stat(fs.f.Name())
		if os.IsNotExist(err) {
			fs.f = nil
		}
	}

	if fs.f == nil {
		return fs.open()
	}

	err := fs.f.Close()
	fs.f = nil
	if err != nil {
		return err
	}

	return fs.open()
}

func (fs *fileSink) rotate() error {
	elapsed := time.Since(fs.lastCreated)
	if (fs.bytesWritten >= int64(fs.maxBytes) && (fs.maxBytes > 0)) ||
		((elapsed > fs.maxDuration) && (fs.maxDuration > 0)) {

		err := fs.f.Close()
		if err != nil {
			return err
		}
		fs.f = nil

		rotateTime := time.Now().UnixNano()
		rotateFileName := fmt.Sprintf(fs.fileNamePattern(), strconv.FormatInt(rotateTime, 10))
		oldPath := filepath.Join(fs.path, fs.fileName)
		newPath := filepath.Join(fs.path, rotateFileName)
		if err := os.Rename(oldPath, newPath); err != nil {
			return fmt.Errorf("failed to rotate log file: %v", err)
		}

		if err := fs.pruneFiles(); err != nil {
			return fmt.Errorf("failed to prune log files: %w", err)
		}

		return fs.open()
	}

	return nil
}

func (fs *fileSink) open() error {
	if fs.f != nil {
		return nil
	}

	if err := os.MkdirAll(fs.path, dirMode); err != nil {
		return err
	}

	createTime := time.Now()
	newFileName := fs.newFileName()
	newFilePath := filepath.Join(fs.path, newFileName)

	var err error
	fs.f, err = os.OpenFile(newFilePath, os.O_APPEND|os.O_WRONLY|os.O_CREATE, fileMode)
	if err != nil {
		return err
	}

	// Reset file related statistics
	fs.lastCreated = createTime
	fs.bytesWritten = 0

	return nil
}

func (fs *fileSink) pruneFiles() error {
	if fs.maxFiles == 0 {
		return nil
	}

	pattern := fs.fileNamePattern()
	globExpression := filepath.Join(fs.path, fmt.Sprintf(pattern, "*"))
	matches, err := filepath.Glob(globExpression)
	if err != nil {
		return err
	}

	sort.Strings(matches)

	stale := len(matches) - fs.maxFiles
	for i := 0; i < stale; i++ {
		if err := os.Remove(matches[i]); err != nil {
			return err
		}
	}
	return nil
}

func (fs *fileSink) fileNamePattern() string {
	ext := filepath.Ext(fs.fileName)
	if ext == "" {
		ext = ".log"
	}

	return strings.TrimSuffix(fs.fileName, ext) + "-%s" + ext
}

func (fs *fileSink) newFileName() string {
	return fs.fileName
}

func (fs *fileSink) rotateEnabled() bool {
	return fs.maxBytes > 0 || fs.maxDuration != 0
}
