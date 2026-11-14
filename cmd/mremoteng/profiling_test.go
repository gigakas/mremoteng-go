package main

import (
	"strings"
	"testing"
)

func TestProfilingAddress_OnlyAcceptsLoopback(t *testing.T) {
	if address, err := profilingAddress(""); err != nil || address != "" {
		t.Errorf("empty = %q, %v; want profiling off", address, err)
	}
	if address, err := profilingAddress(" 127.0.0.1:6060 "); err != nil || address != "127.0.0.1:6060" {
		t.Errorf("loopback = %q, %v", address, err)
	}
	if address, err := profilingAddress(":6060"); err != nil || address != "127.0.0.1:6060" {
		t.Errorf("port only = %q, %v; want it bound to loopback", address, err)
	}
	for _, exposed := range []string{"0.0.0.0:6060", "192.168.22.112:6060", "[::]:6060"} {
		if _, err := profilingAddress(exposed); err == nil || !strings.Contains(err.Error(), "loopback") {
			t.Errorf("%q was accepted; a profiling endpoint must stay local", exposed)
		}
	}
}
