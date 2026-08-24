package control

import (
	"context"
	"errors"
	"testing"
	"time"

	"reasonix/internal/agent"
	"reasonix/internal/event"
)

// A scheduled task that arrives while a turn is in flight must be reported as
// failed, not dispatched: the scheduler records Success from this return value.
func TestSendCtxReportsTurnAlreadyRunning(t *testing.T) {
	sess := agent.NewSession("sys")
	c := New(Options{Runner: appendingRunner{session: sess}})

	release := make(chan struct{})
	defer close(release)
	c.runGuarded(func(ctx context.Context) error {
		<-release
		return nil
	})
	waitRunning(t, c, true)

	err := c.SendCtx(context.Background(), "scheduled task")
	if !errors.Is(err, ErrTurnRunning) {
		t.Fatalf("SendCtx while busy = %v, want ErrTurnRunning", err)
	}
}

// SendCtx is the scheduler's synchronous entry point: when it returns, the turn
// it started has finished.
func TestSendCtxWaitsForTheTurn(t *testing.T) {
	sess := agent.NewSession("sys")
	c := New(Options{Runner: appendingRunner{session: sess}})

	if err := c.SendCtx(context.Background(), "scheduled task"); err != nil {
		t.Fatalf("SendCtx = %v, want nil", err)
	}
	if c.Running() {
		t.Fatal("turn still running after SendCtx returned")
	}
	if n := len(sess.Messages); n == 0 {
		t.Fatal("SendCtx returned before the runner saw the prompt")
	}
}

// The scheduler wraps each task in a 10-minute deadline. That deadline has to
// bound the turn, so a runner that ignores cancellation cannot pin the caller.
func TestSendCtxHonorsCallerDeadline(t *testing.T) {
	sess := agent.NewSession("sys")
	release := make(chan struct{})
	c := New(Options{Runner: blockingRunner{session: sess, release: release}})
	// Release the runner before waiting on the autosaver, or a regression that
	// leaves the turn running hangs the test instead of failing it.
	defer c.autosaveWG.Wait()
	defer close(release)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	start := time.Now()
	err := c.SendCtx(ctx, "scheduled task")
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("SendCtx past deadline = %v, want DeadlineExceeded", err)
	}
	if elapsed := time.Since(start); elapsed > 5*time.Second {
		t.Fatalf("SendCtx blocked %v past its deadline", elapsed)
	}
}

// runGuarded keeps its fire-and-forget contract for interactive frontends: the
// outcome arrives as TurnDone, and a second call while busy is a no-op.
func TestRunGuardedStaysAsynchronous(t *testing.T) {
	sess := agent.NewSession("sys")
	events := make(chan event.Event, 8)
	c := New(Options{
		Runner: appendingRunner{session: sess},
		Sink:   event.FuncSink(func(e event.Event) { events <- e }),
	})

	release := make(chan struct{})
	c.runGuarded(func(ctx context.Context) error {
		<-release
		return nil
	})
	waitRunning(t, c, true)

	var second bool
	c.runGuarded(func(ctx context.Context) error {
		second = true
		return nil
	})
	if second {
		t.Fatal("runGuarded ran a second body while a turn was in flight")
	}
	close(release)

	deadline := time.After(2 * time.Second)
	for {
		select {
		case e := <-events:
			if e.Kind == event.TurnDone {
				return
			}
		case <-deadline:
			t.Fatal("timed out waiting for TurnDone")
		}
	}
}

func waitRunning(t *testing.T, c *Controller, want bool) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if c.Running() == want {
			return
		}
		time.Sleep(2 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for Running()==%v", want)
}
