package datastore

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"path/filepath"
	"sort"
	"strconv"
	"sync"
	"time"
)

type Db struct {
	segments    []*Segment
	segmentSize int64
	dirPath     string
	combining   bool
	mu          sync.Mutex
}

func NewDb(dir string, segmentSize int64) (*Db, error) {
	db := &Db{
		segments:    []*Segment{},
		segmentSize: segmentSize,
		dirPath:     dir,
		combining:   false,
	}
	err := db.recover()
	if err != nil && err != io.EOF {
		return nil, err
	}
	if len(db.segments) == 0 {
		_, err := db.newSegment()
		if err != nil {
			return nil, err
		}
	}
	return db, nil
}

func (db *Db) newSegment() (*Segment, error) {
	name := time.Now().UnixNano()
	segmentPath := filepath.Join(db.dirPath, strconv.FormatInt(name, 10))
	sgm, err := NewSegment(segmentPath, db.segmentSize)
	if err != nil {
		return nil, err
	}
	db.mu.Lock()
	db.segments = append(db.segments, sgm)
	db.mu.Unlock()
	if len(db.segments) >= 3 {
		go db.combine(len(db.segments) - 1)
	}
	return sgm, err
}

func (db *Db) recover() error {
	files, err := ioutil.ReadDir(db.dirPath)
	if err != nil {
		return err
	}
	var segments []string
	for _, file := range files {
		if file.IsDir() {
			continue
		}
		segments = append(segments, file.Name())
	}
	sort.SliceStable(segments, func(i, j int) bool {
		iTime, err := strconv.Atoi(segments[i])
		if err != nil {
			log.Fatal(err)
		}
		jTime, err := strconv.Atoi(segments[j])
		if err != nil {
			log.Fatal(err)
		}
		return time.Unix(0, int64(iTime)).Before(time.Unix(0, int64(jTime)))
	})
	for _, name := range segments {
		path := filepath.Join(db.dirPath, name)
		sgm, err := NewSegment(path, db.segmentSize)
		if err != nil {
			return err
		}
		err = sgm.recover()
		if err != nil && err != io.EOF {
			return err
		}
		db.segments = append(db.segments, sgm)
	}
	return err
}

func (db *Db) Close() error {
	for _, sgm := range db.segments {
		err := sgm.Close()
		if err != nil {
			return err
		}
	}
	return nil
}

func (db *Db) Get(key string) (string, error) {
	db.mu.Lock()
	sgms := db.segments
	db.mu.Unlock()
	for i := len(sgms) - 1; i >= 0; i-- {
		sgm := sgms[i]
		val, err := sgm.Get(key)

		if err == nil {
			return val, nil
		}
		if err == ErrNotFound {
			continue
		}
	}
	return "", ErrNotFound
}

func (db *Db) Put(key, value string) error {
	e := entry{
		key:   key,
		value: value,
	}
	db.mu.Lock()
	currentSegment := db.segments[len(db.segments)-1]
	db.mu.Unlock()
	if currentSegment.IsFull() {
		sgm, err := db.newSegment()
		if err != nil {
			return err
		}
		currentSegment = sgm
	}
	err := currentSegment.Write(e)
	if err != nil {
		return err
	}
	return nil
}

func (db *Db) combine(n int) error {
	db.mu.Lock()
	defer db.mu.Unlock()
	if db.combining {
		return nil
	}
	db.combining = true
	forUpdate := db.segments[0:n]
	data := make(map[string]string)
	for _, sgm := range forUpdate {
		all, err := sgm.GetAll()
		if err != nil {
			return err
		}
		for key, val := range all {
			data[key] = val
		}
	}
	systemSegmentPath := filepath.Join(db.dirPath, "system-segment")
	sgm, err := NewSegment(systemSegmentPath, db.segmentSize)
	if err != nil {
		return err
	}
	db.mu.Unlock()
	for key, val := range data {
		e := entry{
			key:   key,
			value: val,
		}
		sgm.Write(e)
	}
	db.mu.Lock()
	err = sgm.Relocate(forUpdate[len(forUpdate)-1].path)
	if err != nil {
		return err
	}
	segments := append([]*Segment{sgm}, db.segments[n:]...)
	db.segments = segments
	for i := 0; i < len(forUpdate)-1; i++ {
		sgm := forUpdate[i]
		err := sgm.HardRemove()
		sgm = nil
		if err != nil {
			return fmt.Errorf("remove segment error")
		}
	}
	db.combining = false
	return nil
}
