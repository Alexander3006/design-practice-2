package datastore

import (
	"io/ioutil"
	"os"
	"testing"
)

const segmentSize = 10240

func TestDb_Put(t *testing.T) {
	dir, err := ioutil.TempDir(".", "test-db-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	db, err := NewDb(dir, segmentSize)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	pairs := [][]string{
		{"key1", "value1"},
		{"key2", "value2"},
		{"key3", "value3"},
	}

	t.Run("put/get", func(t *testing.T) {
		for _, pair := range pairs {
			res := make(chan *entry)
			err := db.Put(pair[0], pair[1], res)
			<- res
			if err != nil {
				t.Errorf("Cannot put %s: %s", pairs[0], err)
			}
			value, err := db.Get(pair[0])
			if err != nil {
				t.Errorf("Cannot get %s: %s", pairs[0], err)
			}
			if value != pair[1] {
				t.Errorf("Bad value returned expected %s, got %s", pair[1], value)
			}
		}
	})

	t.Run("new db process", func(t *testing.T) {
		db.Close()
		db, err = NewDb(dir, segmentSize)
		if err != nil {
			t.Fatal(err)
		}

		for _, pair := range pairs {
			value, err := db.Get(pair[0])
			if err != nil {
				t.Errorf("Cannot put %s: %s", pairs[0], err)
			}
			if value != pair[1] {
				t.Errorf("Bad value returned expected %s, got %s", pair[1], value)
			}
		}
	})

}
