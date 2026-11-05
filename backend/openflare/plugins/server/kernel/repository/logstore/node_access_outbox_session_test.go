package logstore

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type outboxSessionDriver struct {
	conn *outboxSessionConn
}

func (d outboxSessionDriver) Open(string) (driver.Conn, error) { return d.conn, nil }

type outboxSessionConnector struct{ outboxSessionDriver }

func (c outboxSessionConnector) Connect(context.Context) (driver.Conn, error) {
	return c.conn, nil
}
func (c outboxSessionConnector) Driver() driver.Driver { return c.outboxSessionDriver }

type outboxSessionConn struct {
	driver.Conn
	pending       bool
	closed        bool
	locked        bool
	unlockCalls   int
	unlockFailure bool
	lockFailure   bool
	queryFailure  bool
	ackFailure    bool
	unlockContext error
}

func (c *outboxSessionConn) Close() error { c.closed = true; return nil }
func (c *outboxSessionConn) ExecContext(ctx context.Context, query string, _ []driver.NamedValue) (driver.Result, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if strings.Contains(query, "pg_advisory_lock") {
		c.locked = true
		if c.lockFailure {
			return nil, context.Canceled
		}
	} else if strings.HasPrefix(query, "DELETE") {
		if !c.locked {
			return nil, errors.New("ack without session lock")
		}
		if c.ackFailure {
			return nil, errors.New("ack failed")
		}
		c.pending = false
	}
	return driver.RowsAffected(1), nil
}

type outboxSessionRows struct {
	columns []string
	values  []driver.Value
}

func (r *outboxSessionRows) Columns() []string { return r.columns }
func (r *outboxSessionRows) Close() error      { return nil }
func (r *outboxSessionRows) Next(dest []driver.Value) error {
	if r.values == nil {
		return io.EOF
	}
	copy(dest, r.values)
	r.values = nil
	return nil
}

func (c *outboxSessionConn) QueryContext(ctx context.Context, query string, _ []driver.NamedValue) (driver.Rows, error) {
	if strings.Contains(query, "pg_advisory_unlock") {
		c.unlockCalls++
		c.unlockContext = ctx.Err()
		if c.unlockFailure {
			return nil, errors.New("unlock failed")
		}
		c.locked = false
		return &outboxSessionRows{columns: []string{"unlocked"}, values: []driver.Value{true}}, nil
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if !c.locked || c.queryFailure {
		return nil, errors.New("outbox read failed")
	}
	rows := &outboxSessionRows{columns: []string{"id", "payload", "created_at"}}
	if c.pending {
		rows.values = []driver.Value{int64(1), `{"DeleteIDs":[1]}`, time.Now()}
	}
	return rows, nil
}

func TestOutboxPostgresSingleConnection(t *testing.T) {
	for _, scenario := range []string{"success", "delivery failure", "cancel delivery", "unlock failure", "lock failure", "query failure", "ack failure"} {
		t.Run(scenario, func(t *testing.T) {
			conn := &outboxSessionConn{pending: true, unlockFailure: scenario == "unlock failure", lockFailure: scenario == "lock failure", queryFailure: scenario == "query failure", ackFailure: scenario == "ack failure"}
			pool := sql.OpenDB(outboxSessionConnector{outboxSessionDriver{conn}})
			pool.SetMaxOpenConns(1)
			t.Cleanup(func() { pool.Close() })
			db, err := gorm.Open(postgres.New(postgres.Config{Conn: pool}), &gorm.Config{DisableAutomaticPing: true})
			if err != nil {
				t.Fatal(err)
			}
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()
			err = flushNodeOutbox(ctx, db, func(context.Context, nodeDelivery) error {
				if scenario == "cancel delivery" {
					cancel()
				}
				if scenario == "delivery failure" {
					return errors.New("delivery failed")
				}
				return nil
			})
			if (err != nil) != (scenario != "success") {
				t.Errorf("flushNodeOutbox(%s) error=%v, want error=%t", scenario, err, scenario != "success")
			}
			if scenario == "cancel delivery" && !errors.Is(err, context.Canceled) {
				t.Errorf("flushNodeOutbox(cancel delivery)=%v, want context.Canceled", err)
			}
			wantPending := scenario != "success" && scenario != "unlock failure"
			if conn.pending != wantPending {
				t.Errorf("flushNodeOutbox(%s) pending=%t, want %t", scenario, conn.pending, wantPending)
			}
			wantUnlock := 1
			if scenario == "lock failure" {
				wantUnlock = 0
			}
			if conn.unlockCalls != wantUnlock || conn.unlockContext != nil {
				t.Errorf("flushNodeOutbox(%s) unlock calls/context=%d/%v, want %d/nil", scenario, conn.unlockCalls, conn.unlockContext, wantUnlock)
			}
			wantClosed := scenario == "unlock failure" || scenario == "lock failure"
			if conn.closed != wantClosed {
				t.Errorf("flushNodeOutbox(%s) discarded=%t, want %t", scenario, conn.closed, wantClosed)
			}
			if got := pool.Stats().InUse; got != 0 {
				t.Errorf("flushNodeOutbox(%s) connections in use=%d, want 0", scenario, got)
			}
		})
	}
}

func TestOutboxCancelWhileWaitingForFlush(t *testing.T) {
	nodeOutboxGate <- struct{}{}
	defer func() { <-nodeOutboxGate }()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := flushNodeOutbox(ctx, nil, nil); !errors.Is(err, context.Canceled) {
		t.Errorf("flushNodeOutbox(canceled waiter)=%v, want context.Canceled", err)
	}
}
