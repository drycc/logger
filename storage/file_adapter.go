package storage

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path"
	"strconv"
	"strings"
	"sync"
)

var LogRoot = "/data/logs"

type fileAdapter struct {
	files map[string]*os.File
	mutex sync.Mutex
}

// NewFileAdapter returns an Adapter that uses a file.
func NewFileAdapter() (Adapter, error) {
	return &fileAdapter{files: make(map[string]*os.File)}, nil
}

// Start the storage adapter-- in the case of this implementation, a no-op
func (a *fileAdapter) Start() {
}

// Write adds a log message to to an app-specific log file
func (a *fileAdapter) Write(app string, message string) error {
	// Check first if we might actually have to add to the map of file pointers so we can avoid
	// waiting for / obtaining a lock unnecessarily
	f, ok := a.files[app]
	if !ok {
		// Ensure only one goroutine at a time can be adding a file pointer to the map of file
		// pointers
		a.mutex.Lock()
		defer a.mutex.Unlock()
		f, ok = a.files[app]
		if !ok {
			var err error
			f, err = a.getFile(app)
			if err != nil {
				return err
			}
			a.files[app] = f
		}
	}
	if _, err := f.WriteString(message + "\n"); err != nil {
		return err
	}
	return nil
}

// Read retrieves a specified number of log lines from an app-specific log file
func (a *fileAdapter) Read(app string, lines int) ([]string, error) {
	if lines <= 0 {
		return []string{}, nil
	}
	filePath := a.getFilePath(app)
	exists, err := fileExists(filePath)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, fmt.Errorf("could not find logs for '%s'", app)
	}
	logBytes, err := exec.Command("tail", "-n", strconv.Itoa(lines), filePath).Output()
	if err != nil {
		return nil, err
	}
	logStrs := strings.Split(string(logBytes), "\n")
	return logStrs[:len(logStrs)-1], nil
}

// Make Chan a pipeline to read logs all the time
func (a *fileAdapter) Chan(ctx context.Context, app string, size int) (chan string, error) {
	filePath := a.getFilePath(app)
	exists, err := fileExists(filePath)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, fmt.Errorf("could not find logs for '%s'", app)
	}

	channel := make(chan string, size)
	go func() {
		defer close(channel)
		cmd := exec.Command("tail", "-n", "0", "-f", filePath)
		out, err := cmd.StdoutPipe()
		if err != nil {
			panic(err)
		}
		defer out.Close()
		cmd.Start()
		go func() {
			<-ctx.Done()
			cmd.Process.Kill()
		}()
		scanner := bufio.NewScanner(out)
		scanner.Split(bufio.ScanLines)
		for len(channel) != size && scanner.Scan() {
			line := scanner.Text()
			channel <- line
		}
	}()
	return channel, nil
}

// Destroy deletes stored logs for the specified application
func (a *fileAdapter) Destroy(app string) error {
	// Check first if the map of file pointers even contains the file pointer we want so we can avoid
	// waiting for / obtaining a lock unnecessarily
	f, ok := a.files[app]
	if ok {
		// Ensure no other goroutine is trying to modify the file pointer map while we're trying to
		// clean up
		a.mutex.Lock()
		defer a.mutex.Unlock()
		exists, err := fileExists(f.Name())
		if err != nil {
			return err
		}
		if exists {
			if err := os.Remove(f.Name()); err != nil {
				return err
			}
		}
		delete(a.files, app)
	}
	return nil
}

// Reopen every file referenced by this storage adapter
func (a *fileAdapter) Reopen() error {
	// Ensure no other goroutine is trying to add a file pointer to the map of file pointers while
	// we're trying to clear it out
	a.mutex.Lock()
	defer a.mutex.Unlock()
	a.files = make(map[string]*os.File)
	return nil
}

// Stop the storage adapter-- in the case of this implementation, a no-op
func (a *fileAdapter) Stop() {
}

func (a *fileAdapter) getFile(app string) (*os.File, error) {
	filePath := a.getFilePath(app)
	exists, err := fileExists(filePath)
	if err != nil {
		return nil, err
	}
	// return a new file or the existing file for appending
	var file *os.File
	if exists {
		file, err = os.OpenFile(filePath, os.O_RDWR|os.O_APPEND, 0644)
	} else {
		file, err = os.OpenFile(filePath, os.O_RDWR|os.O_CREATE, 0644)
	}
	return file, err
}

func (a *fileAdapter) getFilePath(app string) string {
	return path.Join(LogRoot, app+".log")
}

func fileExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}
