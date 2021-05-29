package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/Alexander3006/design-practice-2/httptools"
	"github.com/Alexander3006/design-practice-2/signal"
	"github.com/gorilla/mux"
)

var port = flag.Int("port", 8080, "server port")

const confHealthFailure = "CONF_HEALTH_FAILURE"

const dbAddr = "http://db:8070"

func insertCommand() {
	const command = "redstone"
	today := time.Now().Format("02-01-2006")
	url := fmt.Sprintf("%s/db/%s", dbAddr, command)
	body := fmt.Sprintf(`{"value":"%s"}`, today)
	body += "\n"
	req, err := http.Post(url, "application/json", strings.NewReader(body))
	if err != nil {
		log.Fatal(err)
	}
	if req.StatusCode != http.StatusOK {
		log.Fatalf("DB responded with status code %s", req.Status)
	}

}

func main() {
	flag.Parse()
	insertCommand()
	r := mux.NewRouter()

	r.HandleFunc("/health", func(rw http.ResponseWriter, r *http.Request) {
		rw.Header().Set("content-type", "text/plain")
		if failConfig := os.Getenv(confHealthFailure); failConfig == "true" {
			rw.WriteHeader(http.StatusInternalServerError)
			_, _ = rw.Write([]byte("FAILURE"))
		} else {
			rw.WriteHeader(http.StatusOK)
			_, _ = rw.Write([]byte("OK"))
		}
	})

	report := make(Report)

	r.HandleFunc("/api/v1/some-data", func(rw http.ResponseWriter, r *http.Request) {
		key := r.FormValue("key")
		log.Printf("Request with key='%s'", key)

		if key == "" {
			rw.WriteHeader(http.StatusNotFound)
			return
		}

		url := fmt.Sprintf("%s/db/%s", dbAddr, key)

		resp, err := http.Get(url)
		if err == nil {
			for k, values := range resp.Header {
				for _, value := range values {
					rw.Header().Add(k, value)
				}
			}

			log.Println("fwd", resp.StatusCode, resp.Request.URL)
			rw.WriteHeader(resp.StatusCode)
			defer resp.Body.Close()

			_, err = io.Copy(rw, resp.Body)
			if err != nil {
				log.Printf("Failed to write response: %s", err)
			}
		} else {
			log.Printf("Failed to get response from: %s", err)
			rw.WriteHeader(http.StatusServiceUnavailable)
		}
	})

	r.Handle("/report", report)

	h := new(http.ServeMux)

	h.Handle("/", r)

	server := httptools.CreateServer(*port, h)
	server.Start()
	signal.WaitForTerminationSignal()
}
