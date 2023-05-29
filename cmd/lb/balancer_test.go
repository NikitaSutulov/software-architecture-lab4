package main

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestScheme(t *testing.T) {
	*https = true
	assert.Equal(t, "https", scheme(), "Expected scheme to be https")

	*https = false
	assert.Equal(t, "http", scheme(), "Expected scheme to be http")
}

func TestHash(t *testing.T) {
	str := "testString"
	assert.IsType(t, uint32(0), hash(str), "Expected output type to be uint32")
}

func TestLoadBalancer(t *testing.T) {
	path1 := "/some/test/path1"
	path2 := "/some/test/path2"

	for i := range healthyServers {
		healthyServers[i] = true
	}

	firstServerPath1 := chooseServer(path1)
	if firstServerPath1 == "" {
		t.Fatal("No server chosen for path1")
	}

	firstServerPath2 := chooseServer(path2)
	if firstServerPath2 == "" {
		t.Fatal("No server chosen for path2")
	}

	for i := 0; i < 10; i++ {
		serverPath1 := chooseServer(path1)
		if serverPath1 != firstServerPath1 {
			t.Fatalf("Different server chosen on iteration %d for path1. First server: %s, this iteration: %s", i, firstServerPath1, serverPath1)
		} else {
			fmt.Printf("Iteration %d for path1: Server %s chosen (Hash: %d)\n", i, serverPath1, hash(path1))
		}

		serverPath2 := chooseServer(path2)
		if serverPath2 != firstServerPath2 {
			t.Fatalf("Different server chosen on iteration %d for path2. First server: %s, this iteration: %s", i, firstServerPath2, serverPath2)
		} else {
			fmt.Printf("Iteration %d for path2: Server %s chosen (Hash: %d)\n", i, serverPath2, hash(path2))
		}
	}
}
