package ssh

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"github.com/labstack/echo/v4"
	"golang.org/x/sync/errgroup"
	"io"
	"net/http"
	"os"
	"os/signal"
	"path"
	"syscall"
	"tailscale.com/tailcfg"
	"tailscale.com/tsnet"
	"time"
)

type RecorderConfig struct {
	LoginServer string
	StateDir    string
	Dir         string
	AuthKey     string
	Hostname    string
}

type CastHeader struct {
	Timestamp int64                `json:"timestamp"`
	SrcNodeID tailcfg.StableNodeID `json:"srcNodeID"`
}

func Start(ctx context.Context, c RecorderConfig) error {
	ctx = contextWithSigterm(ctx)

	s := &tsnet.Server{
		ControlURL: c.LoginServer,
		Dir:        c.StateDir,
		AuthKey:    c.AuthKey,
		Hostname:   c.Hostname,
	}

	if err := waitTSReady(ctx, s); err != nil {
		return err
	}

	mux := echo.New()
	mux.HideBanner = true
	mux.POST("/record", record(c.Dir))

	ln, err := s.Listen("tcp", ":80")
	if err != nil {
		return err
	}

	g, gCtx := errgroup.WithContext(ctx)

	go func() {
		<-gCtx.Done()
		_ = s.Close()
	}()

	g.Go(func() error {
		if err := http.Serve(ln, mux); err != nil && !errors.Is(err, http.ErrServerClosed) {
			return err
		}
		return nil
	})

	return g.Wait()
}

func waitTSReady(ctx context.Context, s *tsnet.Server) error {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	_, err := s.Up(ctx)
	if err != nil {
		return err
	}

	return nil
}

func contextWithSigterm(ctx context.Context) context.Context {
	ctxWithCancel, cancel := context.WithCancel(ctx)
	go func() {
		defer cancel()

		signalCh := make(chan os.Signal, 1)
		signal.Notify(signalCh, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

		select {
		case <-signalCh:
		case <-ctx.Done():
		}
	}()

	return ctxWithCancel
}

func record(dir string) func(echo.Context) error {
	return func(c echo.Context) error {
		reader := bufio.NewReader(c.Request().Body)

		line, err := reader.ReadBytes('\n')
		if err != nil {
			return err
		}

		var header CastHeader
		if err := json.Unmarshal(line, &header); err != nil {
			return err
		}

		timstamp := time.Unix(header.Timestamp, 0)

		nodeRecordingDir := path.Join(dir, string(header.SrcNodeID))
		nodeRecordingFilePath := path.Join(nodeRecordingDir, timstamp.Format(time.RFC3339)+".cast")

		if err = os.MkdirAll(nodeRecordingDir, 0755); err != nil {
			return err
		}

		f, err := os.OpenFile(nodeRecordingFilePath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
		if err != nil {
			return err
		}

		_, err = f.Write(line)
		if err != nil {
			return err
		}

		if _, err := io.Copy(f, reader); err != nil {
			return err
		}

		return c.String(200, "ok")
	}
}
