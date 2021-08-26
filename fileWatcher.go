package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"math/rand"
	"os"
	"time"
)

// filewatcher utility to get file change events to sse broker channel

type FileWatcher struct {
	// path of file to watch
	filePath string
	// lastModified time of file
	lastMod time.Time
	// position at which the file was last read
	seekCursor int64
	// channel to send messages to
	subscriber func(message string)
}

func NewFileWatcher(filePath string) (*FileWatcher, error) {
	f, err := os.Stat(filePath)
	if err != nil {
		return nil, err
	}

	fw := &FileWatcher{
		filePath:   filePath,
		lastMod:    f.ModTime(),
		seekCursor: f.Size(),
	}
	return fw, nil
}

func (fw *FileWatcher) Subscribe(subscriberChan chan<- string) error {
	fw.subscriber = func(message string) {
		subscriberChan <- message
	}

	// start process to watch file events
	go func() {
		ticker := time.NewTicker(1 * time.Second)
		for i := 0; ; i++ {
			<-ticker.C

			cursorDiff, err := fw.getUpdates()
			if err != nil {
				log.Fatalln("Err while getting file update", err.Error())
			}

			if !cursorDiff {
				continue
			}

			if _, err = fw.getDiff(); err != nil {
				log.Fatalln("Err while fetching diff", err.Error())
			}

			log.Println(cursorDiff, "file modified")
		}
	}()

	return nil
}

func (fw *FileWatcher) getDiff() (string, error) {
	file, err := os.Open(fw.filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	file.Seek(fw.seekCursor, io.SeekStart)
	scanner := bufio.NewScanner(file)

	scanLines := func(data []byte, atEOF bool) (advance int, token []byte, err error) {
		advance, token, err = bufio.ScanLines(data, atEOF)
		fw.seekCursor += int64(advance)
		return
	}

	scanner.Split(scanLines)
	for scanner.Scan() {
		fw.subscriber(fmt.Sprintf("Pos: %d, Scanned: %s\n", fw.seekCursor, scanner.Text()))
	}

	return "", nil
}

func (fw *FileWatcher) getUpdates() (bool, error) {
	fs, err := os.Stat(fw.filePath)
	if err != nil {
		return false, err
	}

	if isMod := fs.ModTime().After(fw.lastMod); isMod {
		// update file mod time and cursor
		fw.lastMod = fs.ModTime()

		return true, nil
	}
	return false, nil
}

// Mock logger to add messages to file
func (fw *FileWatcher) MockLogger() {
	go func() {
		f, err := os.OpenFile(fw.filePath, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
		if err != nil {
			log.Fatalln(err.Error())
		}

		defer f.Close()

		for i := 0; ; i++ {
			if _, err = f.WriteString(fmt.Sprintf("%d - the time is %v\n", i, time.Now())); err != nil {
				log.Fatalln(err)
			}

			time.Sleep(time.Millisecond * (500 + time.Duration(rand.Intn(500))))
		}
	}()
}
