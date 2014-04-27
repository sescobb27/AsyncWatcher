package watch

import (
	"bufio"
	"crypto/md5"
	"errors"
	"fmt"
	"io"
	"os"
	"sync"
	"time"
)

const (
	defaultTime time.Duration = time.Minute // 1min = 1000 m/s * 60s
)

type Watch struct {
	lock  sync.RWMutex
	files map[string]string
	time  time.Duration
}

// Initialize Watcher to watch files every n seconds
func NewWatcher(sec time.Duration) *Watch {
	if sec < (time.Second * 5) {
		sec = defaultTime
	}
	return &Watch{
		files: make(map[string]string),
		time:  sec}
}

func checksum(path string, callback func(content string), onError func(err error)) {
	hash := md5.New()
	file, err := os.Open(path)
	if err != nil {
		onError(err)
		return
	}
	defer file.Close()

	reader := bufio.NewReader(file)
	buffer := make([]byte, 1024)

	for {
		nbytes, err := reader.Read(buffer)
		if err != nil && err != io.EOF {
			onError(err)
			return
		}
		io.WriteString(hash, string(buffer[:nbytes]))
		if nbytes == 0 {
			break
		}
	}
	if err != nil {
		onError(err)
		return
	}
	callback(fmt.Sprintf("%x", hash.Sum(nil)))
}

// verify existence of file, permissions and throw error
// if path is a dir (TO-DO)
func verifyPath(path string) error {
	fileInfo, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			// log no exist
		} else if os.IsPermission(err) {
			//log permission denied
		}
		return err
	}
	if fileInfo.IsDir() {
		// TO-DO
		return errors.New("TO-DO\n")
	}
	return nil
}

// verify the existence of path in the watching map
func exist(path *string, watching map[string]string) bool {
	_, ok := watching[*path]
	return ok
}

func (w *Watch) AddFile(path string, event chan string) error {
	err := verifyPath(path)
	if err != nil {
		return err
	}

	w.lock.RLock()
	if exist(&path, w.files) {
		return errors.New(fmt.Sprintf("Already watching %s\n", path))
	}
	w.lock.RUnlock()

	fmt.Fprintf(os.Stdout, "Start: adding md5 checksum of %s to watcher\n", path)

	addCallback := func(content string) {
		w.lock.Lock()

		w.files[path] = content
		fmt.Fprintf(os.Stdout, "Finish: adding md5 checksum of %s to watcher: %s\n", path, content)

		w.lock.Unlock()
		w.onChange(path, event)
	}

	onError := func(err error) {
		fmt.Fprintf(os.Stderr, "Error: in checksum %s\n", err)
	}
	// each file has its own goroutine and callback, so that we need to have
	// a mutex for access watching files map
	go checksum(path, addCallback, onError)
	return nil
}

func (w *Watch) onChange(path string, event chan string) {
	var wg sync.WaitGroup
	updateAndNotifyCallback := func(content string) {
		w.lock.RLock()
		v, _ := w.files[path]
		if v != content {
			w.lock.RUnlock()
			w.lock.Lock()
			w.files[path] = content
			w.lock.Unlock()
			event <- path
			// fmt.Fprintf(os.Stdout, "Change Event on %s: checksum is: %s\n", path, content)
		} else {
			w.lock.RUnlock()
		}
		wg.Done()
	}
	onError := func(err error) {
		fmt.Fprintf(os.Stderr, "Error: in OnChange %s\n", err)
		wg.Done()
	}
	for {
		wg.Add(1)
		go checksum(path, updateAndNotifyCallback, onError)
		wg.Wait()
		time.Sleep(w.time)
	}
}
