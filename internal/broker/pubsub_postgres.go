package broker

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/lib/pq"
	"github.com/pkg/errors"
	"time"
)

type pgPubsub struct {
	pgListener *pq.Listener
	db         *sql.DB
	target     Pubsub
}

// NewPubsub creates a new Pubsub implementation using a PostgreSQL connection.
func NewPubsub(ctx context.Context, database *sql.DB, connectURL string) (Pubsub, error) {
	// Creates a new listener using pq.
	errCh := make(chan error)
	listener := pq.NewListener(connectURL, time.Second, time.Minute, func(event pq.ListenerEventType, err error) {
		// This callback gets events whenever the connection state changes.
		// Don't send if the errChannel has already been closed.
		select {
		case <-errCh:
			return
		default:
			errCh <- err
			close(errCh)
		}
	})

	select {
	case err := <-errCh:
		if err != nil {
			return nil, errors.Errorf("create pq listener: %w", err)
		}
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	if err := listener.Listen("ionscale_events"); err != nil {
		return nil, errors.Errorf("listen: %w", err)
	}

	pubsub := &pgPubsub{
		db:         database,
		pgListener: listener,
		target:     NewPubsubInMemory(),
	}
	go pubsub.listen(ctx)

	return pubsub, nil
}

// Close closes the pubsub instance.
func (p *pgPubsub) Close() error {
	return p.pgListener.Close()
}

func (p *pgPubsub) Subscribe(tailnet uint64, listener Listener) (cancel func(), err error) {
	return p.target.Subscribe(tailnet, listener)
}

func (p *pgPubsub) Publish(tailnet uint64, message *Signal) error {
	event := &pgEvent{
		TailnetID: tailnet,
		Signal:    message,
	}

	payload, err := json.Marshal(event)
	if err != nil {
		return err
	}

	// This is safe because we are calling pq.QuoteLiteral. pg_notify doesn't
	// support the first parameter being a prepared statement.
	//nolint:gosec
	_, err = p.db.ExecContext(context.Background(), `select pg_notify(`+pq.QuoteLiteral("ionscale_events")+`, $1)`, payload)
	if err != nil {
		fmt.Println(err)
		return errors.Errorf("exec pg_notify: %w", err)
	}
	return nil
}

// listen begins receiving messages on the pq listener.
func (p *pgPubsub) listen(ctx context.Context) {
	var (
		notif *pq.Notification
		ok    bool
	)
	defer p.pgListener.Close()
	for {
		select {
		case <-ctx.Done():
			return
		case notif, ok = <-p.pgListener.Notify:
			if !ok {
				return
			}
		}
		// A nil notification can be dispatched on reconnect.
		if notif == nil {
			continue
		}
		p.listenReceive(notif)
	}
}

func (p *pgPubsub) listenReceive(notif *pq.Notification) {
	extra := []byte(notif.Extra)
	event := &pgEvent{}

	if err := json.Unmarshal(extra, event); err == nil {
		p.target.Publish(event.TailnetID, event.Signal)
	} else {
		fmt.Println(err)
	}
}

type pgEvent struct {
	TailnetID uint64
	Signal    *Signal
}
