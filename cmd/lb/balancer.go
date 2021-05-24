package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"time"

	"github.com/Alexander3006/design-practice-2/httptools"
	"github.com/Alexander3006/design-practice-2/signal"
)

var (
	port         = flag.Int("port", 8090, "load balancer port")
	timeoutSec   = flag.Int("timeout-sec", 3, "request timeout time in seconds")
	https        = flag.Bool("https", false, "whether backends support HTTPs")
	traceEnabled = flag.Bool("trace", false, "whether to include tracing information into responses")
)

var (
	timeout     = time.Duration(*timeoutSec) * time.Second
	serversPool = []string{
		"server1:8080",
		"server2:8080",
		"server3:8080",
	}
)

func scheme() string {
	if *https {
		return "https"
	}
	return "http"
}

type Backend struct {
	URL    string
	Alive  bool
	Weight int
}

func NewPool(urls ...string) []*Backend {
	var servers []*Backend
	for _, url := range urls {
		servers = append(servers, &Backend{URL: url})
	}
	return servers
}

func FirstHealthServer(servers []*Backend) int {
	for i, s := range servers {
		if s.Alive {
			return i
		}
		continue
	}
	return -1
}

func Min(servers []*Backend) int {
	index := FirstHealthServer(servers)
	if index == -1 {
		return index
	}
	min := servers[index].Weight
	for i := index; i < len(servers); i++ {
		if servers[i].Weight < min {
			if servers[i].Alive {
				index = i
				min = servers[i].Weight
			}
		}
		continue
	}

	return index
}

func getServer(servers []*Backend) (*Backend, error) {
	serverIndex := Min(servers)
	if serverIndex == -1 {
		return nil, errors.New("No healthy servers")
	}
	server := servers[serverIndex]
	return server, nil
}

func forward(servers []*Backend, rw http.ResponseWriter, r *http.Request) error {
	server, err := getServer(servers)
	if err != nil {
		log.Println(err)
		return err
	}

	dst := server.URL

	ctx, _ := context.WithTimeout(r.Context(), timeout)
	fwdRequest := r.Clone(ctx)
	fwdRequest.RequestURI = ""
	fwdRequest.URL.Host = dst
	fwdRequest.URL.Scheme = scheme()
	fwdRequest.Host = dst

	resp, err := http.DefaultClient.Do(fwdRequest)
	if err == nil {
		for k, values := range resp.Header {
			for _, value := range values {
				rw.Header().Add(k, value)
			}
		}
		if *traceEnabled {
			rw.Header().Set("lb-from", dst)
		}
		log.Println("fwd", resp.StatusCode, resp.Request.URL)
		rw.WriteHeader(resp.StatusCode)

		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Printf("Error reading from the server: %s", err)
			return err
		}
		add := len(body)
		server.Weight += add
		log.Printf(`Read "%s"`, string(body))
		_, err = rw.Write(body)
		if err != nil {
			log.Printf("Failed to write response: %s", err)
			return err
		}
		return nil
	} else {
		log.Printf("Failed to get response from %s: %s", dst, err)
		rw.WriteHeader(http.StatusServiceUnavailable)
		return err
	}
}

func health(dst string) bool {
	ctx, _ := context.WithTimeout(context.Background(), timeout)
	req, _ := http.NewRequestWithContext(ctx, "GET",
		fmt.Sprintf("%s://%s/health", scheme(), dst), nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return false
	}
	if resp.StatusCode != http.StatusOK {
		return false
	}
	return true
}

func main() {
	flag.Parse()

	servers := NewPool(serversPool...)
	for _, server := range servers {
		server := server
		go func() {
			for range time.Tick(10 * time.Second) {
				alive := health(server.URL)
				log.Println(server.URL, alive)
				server.Alive = alive
			}
		}()
	}

	frontend := httptools.CreateServer(*port, http.HandlerFunc(
		func(rw http.ResponseWriter, r *http.Request) {
			forward(servers, rw, r)
		}))

	log.Println("Starting load balancer...")
	log.Printf("Tracing support enabled: %t", *traceEnabled)
	frontend.Start()
	signal.WaitForTerminationSignal()
}
