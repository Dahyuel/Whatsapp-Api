package fingerprint

import (
	"math/rand"
	"time"
)

// NetworkJitter returns a random latency duration between minMs and maxMs milliseconds.
// This simulates realistic network variation to make sessions appear as different physical devices.
func NetworkJitter(minMs, maxMs int) time.Duration {
	if maxMs <= minMs {
		return time.Duration(minMs) * time.Millisecond
	}
	ms := minMs + rand.Intn(maxMs-minMs)
	return time.Duration(ms) * time.Millisecond
}

// SimulateNetworkDelay sleeps for a realistic randomized network latency.
// Default range: 50–300ms.
func SimulateNetworkDelay() {
	time.Sleep(NetworkJitter(50, 300))
}
