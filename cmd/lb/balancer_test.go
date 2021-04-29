package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBalancer(t *testing.T) {
	TestCases := []struct {
		ServerPool []*Backend
		Server     *Backend
	}{
		{
			ServerPool: []*Backend{
				{
					URL:    "server1:8080",
					Alive:  true,
					Weight: 150,
				},
				{
					URL:    "server2:8080",
					Alive:  true,
					Weight: 50,
				},
				{
					URL:    "server3:8080",
					Alive:  true,
					Weight: 100,
				},
			},
			Server: &Backend{
				URL:    "server2:8080",
				Alive:  true,
				Weight: 50,
			},
		},
		{
			ServerPool: []*Backend{
				{
					URL:    "server1:8080",
					Alive:  true,
					Weight: 150,
				},
				{
					URL:    "server2:8080",
					Alive:  false,
					Weight: 50,
				},
				{
					URL:    "server3:8080",
					Alive:  true,
					Weight: 100,
				},
			},
			Server: &Backend{
				URL:    "server3:8080",
				Alive:  true,
				Weight: 100,
			},
		},
		{
			ServerPool: []*Backend{
				{
					URL:    "server1:8080",
					Alive:  true,
					Weight: 150,
				},
				{
					URL:    "server2:8080",
					Alive:  false,
					Weight: 50,
				},
				{
					URL:    "server3:8080",
					Alive:  false,
					Weight: 100,
				},
			},
			Server: &Backend{
				URL:    "server1:8080",
				Alive:  true,
				Weight: 150,
			},
		},
		{
			ServerPool: []*Backend{
				{
					URL:    "server1:8080",
					Alive:  false,
					Weight: 150,
				},
				{
					URL:    "server2:8080",
					Alive:  true,
					Weight: 50,
				},
				{
					URL:    "server3:8080",
					Alive:  true,
					Weight: 100,
				},
			},
			Server: &Backend{
				URL:    "server2:8080",
				Alive:  true,
				Weight: 50,
			},
		},
		{
			ServerPool: []*Backend{
				{
					URL:    "server1:8080",
					Alive:  true,
					Weight: 100,
				},
				{
					URL:    "server2:8080",
					Alive:  true,
					Weight: 100,
				},
				{
					URL:    "server3:8080",
					Alive:  true,
					Weight: 100,
				},
			},
			Server: &Backend{
				URL:    "server1:8080",
				Alive:  true,
				Weight: 100,
			},
		},
		{
			ServerPool: []*Backend{
				{
					URL:    "server1:8080",
					Alive:  false,
					Weight: 100,
				},
				{
					URL:    "server2:8080",
					Alive:  false,
					Weight: 100,
				},
				{
					URL:    "server3:8080",
					Alive:  false,
					Weight: 100,
				},
			},
			Server: nil,
		},
		{
			ServerPool: []*Backend{},
			Server:     nil,
		},
		{
			ServerPool: nil,
			Server:     nil,
		},
	}

	for _, c := range TestCases {
		res, _ := getServer(c.ServerPool)
		assert.Equal(t, c.Server, res)
	}
}
