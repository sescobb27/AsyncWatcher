package watch

import (
	"os"
	"testing"
	"time"
)

func assertNoError(err error, t *testing.T) {
	if err != nil {
		t.Error(err)
	}
}

func assertTrue(assertion bool, t *testing.T, msg string) {
	if !assertion {
		t.Error(msg)
	}
}

func assertSignal(ch_signal chan string, t *testing.T, signal_msg string) {
	path, done := <-ch_signal
	if !done && path != signal_msg {
		t.Error("Error: it should send a signal with the changed file path")
	}
}

func assertNoSignal(ch_signal chan string, t *testing.T) {
	_, done := <-ch_signal
	if done {
		t.Error("Error: it should no send a signal")
	}
}

func TestWatcherNoChanges(t *testing.T) {
	watcher := NewWatcher(5 * time.Second)
	file := "testfile.txt"
	event := make(chan string)
	err := watcher.AddFile(file, event)
	assertNoError(err, t)
	go assertNoSignal(event, t)
	<-time.After(6 * time.Second)
	close(event)
}

func TestWatcherChanges(t *testing.T) {
	watcher := NewWatcher(5 * time.Second)
	file := "testfile.txt"
	event := make(chan string)
	err := watcher.AddFile(file, event)
	assertNoError(err, t)
	go assertSignal(event, t, file)

	<-time.After(5 * time.Second)
	os_filed, err := os.OpenFile(file, os.O_WRONLY|os.O_APPEND, 0777)
	assertNoError(err, t)
	os_filed.WriteString("Hello World Golang Watcher\n")
	os_filed.Close()
}
