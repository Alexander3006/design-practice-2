package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
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
	timeout = time.Duration(*timeoutSec) * time.Second
	servers = []string{
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

type Backend struct {
	URL    *url.URL
	Alive  bool
	Weight int
}

type LoadBalance struct {
	backends []*Backend
}

func newLoadBalance() *LoadBalance {
	return &LoadBalance{}
}

func (l *LoadBalance) AddServers(urls ...string) {
	for _, rawUrl := range urls {
		u, _ := url.Parse(rawUrl)
		l.backends = append(l.backends, &Backend{URL: u})
	}
}

func (l *LoadBalance) FirstHealthServer() int {
	for i, b := range l.backends {
		if b.Alive {
			return i
		}
		continue
	}
	return -1
}

func (l *LoadBalance) Min() (int, error) {
	index := l.FirstHealthServer()
	if index == -1 {
		return index, errors.New("No healthy servers")
	}
	servers := l.backends
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

	return index, nil
}

func (l *LoadBalance) forward(rw http.ResponseWriter, r *http.Request) error {
	serverIndex, err := l.Min()
	if err != nil {
		log.Println(err)
		return err
	}

	server := l.backends[serverIndex]
	dst := server.URL.String()

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
		defer resp.Body.Close()

		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Fatalln(err)
		}
		add := len(body)
		server.Weight += add

		_, err = io.Copy(rw, resp.Body)
		if err != nil {
			log.Printf("Failed to write response: %s", err)
		}
		return nil
	} else {
		log.Printf("Failed to get response from %s: %s", dst, err)
		rw.WriteHeader(http.StatusServiceUnavailable)
		return err
	}
}

func main() {
	flag.Parse()

	lb := newLoadBalance()
	lb.AddServers(servers...)
	for _, server := range lb.backends {
		server := server
		go func() {
			for range time.Tick(10 * time.Second) {
				alive := health(server.URL.String())
				log.Println(server.URL, alive)
				server.Alive = alive
			}
		}()
	}

	frontend := httptools.CreateServer(*port, http.HandlerFunc(
		func(rw http.ResponseWriter, r *http.Request) {
			lb.forward(rw, r)
		}))

	log.Println("Starting load balancer...")
	log.Printf("Tracing support enabled: %t", *traceEnabled)
	frontend.Start()
	signal.WaitForTerminationSignal()
}
