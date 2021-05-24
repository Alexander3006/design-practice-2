package integration

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math"
	"net/http"
	"testing"
	"time"
)

const baseAddress = "http://balancer:8090"

var client = http.Client{
	Timeout: 3 * time.Second,
}

type Response struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// Finds the server with minimum number of bytes
// Returns the triplet of (empty, minName, minBytes)
// If the map is empty, first argument 'empty' will be true
func min(servers map[string]int64) (bool, string, int64) {
	if len(servers) == 0 {
		return true, "", 0
	}
	min := int64(math.MaxInt64)
	minName := ""
	for name, bytes := range servers {
		if bytes < min {
			min = bytes
			minName = name
		}
	}
	return false, minName, min
}

func TestBalancer(t *testing.T) {
	const REQUEST_NUM = 100
	// we store map with ${server name} -> ${bytes received}
	servers := make(map[string]int64)

	for i := 0; i < REQUEST_NUM; i++ {
		resp, err := client.Get(fmt.Sprintf("%s/api/v1/some-data?key=redstone", baseAddress))
		if err != nil {
			t.Error(err)
		}
		server := resp.Header.Get("lb-from")
		bytes := resp.ContentLength

		// lookup server with minimal number of bytes
		empty, minName, minBytes := min(servers)

		// we expect that the responce server has less or equal to minimum number of bytes
		// so, if there is the server which send less number of bytes than the one that
		// we received response from, it's clearly an error
		if !empty && servers[server] > minBytes {
			t.Errorf("expected %s with %d bytes, but got %s with %d bytes", minName, minBytes, server, servers[server])
		}

		servers[server] += bytes

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("Status code %s", resp.Status)
		}

		raw, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			t.Fatalf("Error reading response: %s", err)
		}
		var body Response
		err = json.Unmarshal(raw, &body)
		if err != nil {
			t.Fatalf("Error decoding json: %s (%s)", err, string(raw))
		}
		today := time.Now().Format("01-01-2001")
		if body.Value != today {
			t.Fatalf("Value returned from db is wrong: expected %s but got %s", today, body.Value)
		}
	}

}

func BenchmarkBalancer(b *testing.B) {
	for n := 0; n < b.N; n++ {
		_, err := client.Get(fmt.Sprintf("%s/api/v1/some-data?key=redstone", baseAddress))
		if err != nil {
			b.Error(err)
		}
	}
}
