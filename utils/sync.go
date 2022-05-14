package utils

import (
	"fmt"
	"sync"
	"time"
)

// Waits for the lock for the specified max timeout.
// Returns false if waiting timed out.
func LockTimeout(mx *sync.Mutex, timeout time.Duration) bool {
	c := make(chan struct{})
	go func() {
		defer close(c)
		mx.Lock()
	}()
	select {
	case <-c:
		return true // completed normally
	case <-time.After(timeout):
		return false // timed out
	}
}

// Waits for the waitgroup for the specified max timeout.
// Returns false if waiting timed out.
func WaitTimeout(wg *sync.WaitGroup, timeout time.Duration) bool {
	c := make(chan struct{})
	go func() {
		defer close(c)
		wg.Wait()
	}()
	select {
	case <-c:
		return true // completed normally
	case <-time.After(timeout):
		fmt.Println("==> timedout")
		return false // timed out
	}
}
