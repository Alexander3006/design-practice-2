package datastore

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"
	"os"
)

type hashIndex map[string]int64
var ErrNotFound = fmt.Errorf("record does not exist")

type Segment struct {
	path string
	file *os.File
	outOffset int64
	maxSize int64
	index hashIndex
}

func NewSegment(path string, maxSize int64) (*Segment, error) {
	f, err := os.OpenFile(path, os.O_APPEND|os.O_RDWR|os.O_CREATE, 0o600)
	if err != nil {
		return nil, err
	}
	sgm := &Segment{
		path: path,
		file: f,
		outOffset: 0,
		maxSize: maxSize,
		index: map[string]int64{},
	}
	return sgm, nil
}

func (sgm Segment) IsFull() bool {
	return sgm.outOffset >= sgm.maxSize
}

func (sgm* Segment) Write(data entry) error {
	n, err := sgm.file.Write(data.Encode())
	if err != nil {
		return err
	}
	sgm.index[data.key] = sgm.outOffset
	sgm.outOffset += int64(n)
	return nil
}

func (sgm Segment) Close() error {
	err := sgm.file.Close()
	return err
}

const bufSize = 8192

func (sgm *Segment) recover() error {
	input := sgm.file
	var buf [bufSize]byte
	in := bufio.NewReaderSize(input, bufSize)
	var err error = nil
	for err == nil {
		var (
			header, data []byte
			n int
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

func (sgm Segment) Get(key string) (string, error) {
		position, ok := sgm.index[key]
		if !ok {
			return "", ErrNotFound
		}

		file := sgm.file

		_, err := file.Seek(position, 0)
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
	for key, _ := range sgm.index {
		val, err := sgm.Get(key)
		if err != nil {
			return nil, err
		}
		all[key] = val
	}
	return all, nil
}

func (sgm *Segment) Relocate(path string) error {
	err := sgm.file.Close()
	if err != nil {
		return err
	}
	err = os.Rename(sgm.path, path)
	if err != nil {
		return err
	}
	file, err := os.OpenFile(path, os.O_APPEND|os.O_RDWR|os.O_CREATE, 0o600)
	if err != nil {
		return err
	}
	sgm.file = file
	sgm.path = path
	return nil
}

func (sgm *Segment) HardRemove() error {
	err := os.Remove(sgm.path)
	return err
}