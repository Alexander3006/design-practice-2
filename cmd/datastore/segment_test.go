package datastore

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
)

// We want to crate 2 segments - minimum number before segments start to merge.
// To do this, we set the segment to be 1 KB and write 2 KB of entries.
// Our keys and values both have length of 10 bytes. Together with an
// entry header it should be 32 bytes per entry.
const KB = 1024
const ENTRY = 32
const ENTRY_NUMBER = 2 * KB / ENTRY

// Craft 10 bytes wide string
func craft_string(i int) string {
	return fmt.Sprintf("%010d", i)
}

func Test_Segment(t *testing.T) {
	dir, err := ioutil.TempDir(".", "test-segment-*")
	if err != nil {
		t.Fatal("Creating db store error")
	}
	defer os.RemoveAll(dir)

	db, err := NewDb(dir, KB)
	if err != nil {
		t.Fatal("Creating db error", err)
	}

	for i := 0; i < ENTRY_NUMBER; i++ {
		key := craft_string(i)
		err := db.Put(key, key)
		if err != nil {
			t.Errorf("Put db error: %d", i)
		}
	}

	for i := 0; i < ENTRY_NUMBER; i++ {
		key := craft_string(i)
		val, err := db.Get(key)
		if err != nil {
			t.Errorf("Get db error: %d", i)
		}
		if val != key {
			t.Errorf("Compare val error: %d", i)
		}
	}

	filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			t.Errorf("Walking error: %s", err)
		}
		// skip "."
		if info.IsDir() {
			return nil
		}
		if info.Size() > KB {
			t.Errorf("segment %s: size is %d, but expected to be less than %d", path, info.Size(), KB)
		}
		return nil
	})
}
