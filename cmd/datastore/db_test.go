package datastore

import (
	"io/ioutil"
	"os"
	"testing"
)

const segmentSize = 10240

var pairs = [][]string{
	{"key1", "value1"},
	{"key2", "value2"},
	{"key3", "value3"},
}

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

	t.Run("put/get", func(t *testing.T) {
		for _, pair := range pairs {
			err := db.Put(pair[0], pair[1])
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

func TestDb_Delete(t *testing.T) {
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

	t.Run("delete/get", func(t *testing.T) {
		for _, pair := range pairs {
			err := db.Put(pair[0], pair[1])
			if err != nil {
				t.Errorf("Cannot put %s: %s", pairs[0], err)
			}
			err = db.Delete(pair[0])
			if err != nil {
				t.Errorf("Cannot delete %s: %s", pairs[0], err)
			}
			_, err = db.Get(pair[0])
			if err == nil {
				t.Errorf("Value not delete %s", pairs[0])
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
			_, err := db.Get(pair[0])
			if err == nil {
				t.Errorf("Cannot delete %s", pairs[0])
			}
		}
	})
}
