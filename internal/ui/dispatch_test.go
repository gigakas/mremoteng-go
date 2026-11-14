package ui

import (
	"os"
	"sync"
	"testing"
	"time"
)

// uiQueue holds the callbacks workers posted through runOnUI until the test
// goroutine runs them, standing in for the Fyne main loop.
var uiQueue struct {
	mu    sync.Mutex
	tasks []func()
}

func postToTestUI(f func()) {
	uiQueue.mu.Lock()
	uiQueue.tasks = append(uiQueue.tasks, f)
	uiQueue.mu.Unlock()
}

// drainUI runs every queued callback on the calling (test) goroutine, including
// the ones those callbacks queue in turn.
func drainUI() {
	for {
		uiQueue.mu.Lock()
		tasks := uiQueue.tasks
		uiQueue.tasks = nil
		uiQueue.mu.Unlock()
		if len(tasks) == 0 {
			return
		}
		for _, task := range tasks {
			task()
		}
	}
}

func TestMain(m *testing.M) {
	runOnUI = postToTestUI
	os.Exit(m.Run())
}

// waitFor polls until check passes or the deadline expires, running the queued
// UI callbacks between attempts: workers run on their own goroutines, so the
// effect of a session ending is not synchronous.
func waitFor(t *testing.T, check func() bool) bool {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		drainUI()
		if check() {
			return true
		}
		time.Sleep(time.Millisecond)
	}
	return false
}
