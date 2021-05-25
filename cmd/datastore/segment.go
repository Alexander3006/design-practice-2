package datastore

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"sync"
)

type hashIndex map[string]int64

var ErrNotFound = fmt.Errorf("record does not exist")

type Segment struct {
	path      string
	active    bool
	outOffset int64
	maxSize   int64
	index     hashIndex
	mu        sync.Mutex
	writeChan chan entry
}

func NewSegment(path string, maxSize int64, active bool) (*Segment, error) {
	f, err := os.OpenFile(path, os.O_APPEND|os.O_RDWR|os.O_CREATE, 0o600)
	if err != nil {
		return nil, err
	}
	f.Close()
	sgm := &Segment{
		path:      path,
		active:    active,
		outOffset: 0,
		maxSize:   maxSize,
		index:     map[string]int64{},
	}
	if active {
		writeChan := make(chan entry)
		sgm.writeChan = writeChan
		go sgm.initWritingThread(writeChan)
	}
	return sgm, nil
}

func (sgm *Segment) Write(data entry) error {
	wChan := sgm.writeChan
	if wChan == nil {
		return fmt.Errorf("Can't write to legacy segment")
	}
	wChan <- data
	return nil
}

func (sgm Segment) Close() error {
	return nil
}

const bufSize = 8192

func (sgm *Segment) recover() error {
	input, err := os.Open(sgm.path)
	if err != nil {
		return err
	}
	defer input.Close()

	var buf [bufSize]byte
	in := bufio.NewReaderSize(input, bufSize)
	for err == nil {
		var (
			header, data []byte
			n            int
		)
		header, err = in.Peek(bufSize)
		if err == io.EOF {
			if len(header) == 0 {
				return err
			}
		} else if err != nil {
			return err
		}
		size := binary.LittleEndian.Uint32(header)

		if size < bufSize {
			data = buf[:size]
		} else {
			data = make([]byte, size)
		}
		n, err = in.Read(data)

		if err == nil {
			if n != int(size) {
				return fmt.Errorf("corrupted file")
			}

			var e entry
			e.Decode(data)
			sgm.index[e.key] = sgm.outOffset
			sgm.outOffset += int64(n)
		}
	}
	return err
}

func (sgm *Segment) Get(key string) (string, error) {
	sgm.mu.Lock()
	position, ok := sgm.index[key]
	sgm.mu.Unlock()
	if !ok {
		return "", ErrNotFound
	}

	file, err := os.Open(sgm.path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	_, err = file.Seek(position, 0)
	if err != nil {
		return "", err
	}

	reader := bufio.NewReader(file)
	value, err := readValue(reader)
	if err != nil {
		return "", err
	}
	return value, nil
}

func (sgm Segment) GetAll() (map[string]string, error) {
	all := make(map[string]string)
	for key := range sgm.index {
		val, err := sgm.Get(key)
		if err != nil {
			return nil, err
		}
		all[key] = val
	}
	return all, nil
}

func (sgm *Segment) Relocate(path string) error {
	err := os.Rename(sgm.path, path)
	if err != nil {
		return err
	}
	sgm.path = path
	return nil
}

func (sgm *Segment) HardRemove() error {
	err := os.Remove(sgm.path)
	return err
}

func (sgm *Segment) initWritingThread(writeChan chan entry) error {
	file, err := os.OpenFile(sgm.path, os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	defer file.Close()
	for {
		data, opened := <-writeChan
		if !opened {
			break
		}
		sgm.mu.Lock()
		n, err := file.Write(data.Encode())
		if err != nil {
			return err
		}
		sgm.index[data.key] = sgm.outOffset
		sgm.outOffset += int64(n)
		sgm.active = sgm.outOffset < sgm.maxSize
		sgm.mu.Unlock()
	}
	return nil
}

func (sgm *Segment) StopWritingThread() {
	sgm.writeChan = nil
	close(sgm.writeChan)
}
