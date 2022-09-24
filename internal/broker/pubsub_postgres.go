package broker

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/lib/pq"
	"time"
)

type pgPubsub struct {
	pgListener *pq.Listener
	db         *sql.DB
	target     Pubsub
}

func NewPubsub(ctx context.Context, database *sql.DB, connectURL string) (Pubsub, error) {
	errCh := make(chan error)
	listener := pq.NewListener(connectURL, time.Second, time.Minute, func(event pq.ListenerEventType, err error) {
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
			return nil, fmt.Errorf("create pq listener: %w", err)
		}
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	if err := listener.Listen("ionscale_events"); err != nil {
		return nil, fmt.Errorf("listen: %w", err)
	}

	pubsub := &pgPubsub{
		db:         database,
		pgListener: listener,
		target:     NewPubsubInMemory(),
	}
	go pubsub.listen(ctx)

	return pubsub, nil
}

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

	_, err = p.db.ExecContext(context.Background(), `select pg_notify(`+pq.QuoteLiteral("ionscale_events")+`, $1)`, payload)
	if err != nil {
		return fmt.Errorf("exec pg_notify: %w", err)
	}

	return nil
}

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
