package main

import (
	"encoding/json"
	"flag"
	"log"
	"net/http"
	"os"

	"github.com/Alexander3006/design-practice-2/cmd/datastore"
	"github.com/Alexander3006/design-practice-2/httptools"
	"github.com/Alexander3006/design-practice-2/signal"
	"github.com/gorilla/mux"
)

const KB = 1024
const MB = KB * 1024

var port = flag.Int("p", 8080, "server's port")
var path = flag.String("d", ".db", "database's directory path")
var segment_size = flag.Int("s", 10*MB, "segment size in bytes")

type getResponse struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type postRequest struct {
	Value string `json:"value"`
}

func main() {
	flag.Parse()

	err := os.MkdirAll(*path, os.ModePerm)
	if err != nil {
		log.Fatalf("error creating db directory: %s", err)
		return
	}

	db, err := datastore.NewDb(*path, int64(*segment_size))
	if err != nil {
		log.Fatalf("error creating db: %s", err)
		return
	}
	defer db.Close()
	_ = db.Put("key", "G1gg1L3s")

	r := mux.NewRouter()
	r.HandleFunc("/db/{key}", func(rw http.ResponseWriter, r *http.Request) {
		log.Printf("Get request to %s", r.URL)
		vars := mux.Vars(r)
		key := vars["key"]

		value, err := db.Get(key)

		rw.Header().Set("content-type", "application/json")

		if err != nil {
			rw.WriteHeader(http.StatusNotFound)
		} else {

			rw.WriteHeader(http.StatusOK)

			res := getResponse{key, value}
			err := json.NewEncoder(rw).Encode(&res)

			if err != nil {
				log.Printf("Error while serving request: %s", err)
			}
		}

	}).Methods("GET")

	r.HandleFunc("/db/{key}", func(rw http.ResponseWriter, r *http.Request) {
		log.Printf("Post request to %s", r.URL)
		vars := mux.Vars(r)
		key := vars["key"]

		rw.Header().Set("content-type", "application/json")

		var body postRequest
		err := json.NewDecoder(r.Body).Decode(&body)
		if err != nil {
			rw.WriteHeader(http.StatusBadRequest)
			return
		}

		err = db.Put(key, body.Value)
		if err != nil {
			rw.WriteHeader(http.StatusInternalServerError)
		} else {
			rw.WriteHeader(http.StatusOK)
		}

	}).Methods("POST")

	h := new(http.ServeMux)

	h.Handle("/", r)

	server := httptools.CreateServer(*port, h)
	server.Start()
	signal.WaitForTerminationSignal()
}
