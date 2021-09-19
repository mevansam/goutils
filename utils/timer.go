package utils

import (
	"context"
	"time"
)

// An execution timer will execute a call back
// after a specified period of time. The call
// back can inform the timer to resume to trigger
// at the same interval by returning 0 or update
// the interval by returning the next interval
// period in milliseconds. When the timer first
// starts it invokes the call back to determin
// the trigger interval.
type ExecTimer struct {
	ctx  context.Context
	stop chan bool

	callback     func() (time.Duration, error)
	timeoutTimer *time.Ticker
	
	stopOnError   bool
	callbackError error
}

func NewExecTimer(
	ctx context.Context, 
	callback func() (time.Duration, error),
	stopOnError bool,
) *ExecTimer {
	
	return &ExecTimer{
		ctx:  ctx,
		stop: make(chan bool),

		callback:    callback,
		stopOnError: stopOnError,
	}
}

func (t *ExecTimer) Start(timeout time.Duration) error {

	if timeout == 0 {
		t.invokeCallback()
		return t.callbackError
	}	
	go t.startAuthTimer(timeout)
	return nil
}

func (t *ExecTimer) invokeCallback() bool {
	var (
		err     error
		timeout time.Duration
	)

	if timeout, err = t.callback(); err == nil && timeout == 0 {	
		return false
	}
	if t.timeoutTimer != nil {
		t.timeoutTimer.Stop()
		t.timeoutTimer = nil
	}
	if err != nil {
		t.callbackError = err
	} else {
		go t.startAuthTimer(timeout)
	}
	// returns true to exit the timer loop as a 
	// either a new timeout has been set or callback 
	// returned an error
	return true
}

func (t *ExecTimer) startAuthTimer(timeout time.Duration) {
	
	t.timeoutTimer = time.NewTicker(timeout * time.Millisecond)
	for {
		select {
		case <-t.ctx.Done():
			t.callbackError = t.ctx.Err()
			return
		case <-t.stop:
			return
		case <-t.timeoutTimer.C:			
			if t.invokeCallback() {
				return
			}
		}
	}	
}

func (t *ExecTimer) Stop() error {
	if t.timeoutTimer != nil {
		t.timeoutTimer.Stop()
		t.timeoutTimer = nil

		// wait for current timer to stop
		t.stop <-true
	}
	return t.callbackError
}
