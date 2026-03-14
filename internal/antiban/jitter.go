package antiban

import (
	"math/rand"
	"time"
)

// JitterDelay computes a randomized delay: base +/- a random value in [jitterMin, jitterMax].
// This prevents WhatsApp from detecting deterministic send patterns.
func JitterDelay(base, jitterMin, jitterMax time.Duration) time.Duration {
	if jitterMax < jitterMin {
		jitterMax = jitterMin
	}
	jitter := jitterMin + time.Duration(rand.Int63n(int64(jitterMax-jitterMin+1)))
	return base + jitter
}

// RandomBurstPattern simulates a human-like burst: several messages followed by a longer pause.
// Returns a list of delays for each message in a burst of `count` messages.
func RandomBurstPattern(count int, base, jitterMin, jitterMax time.Duration) []time.Duration {
	delays := make([]time.Duration, count)
	for i := 0; i < count; i++ {
		d := JitterDelay(base, jitterMin, jitterMax)
		// Every 2–3 messages add a longer pause to simulate human conversation rhythm.
		if i > 0 && i%rand.Intn(2+1)+2 == 0 {
			d += time.Duration(rand.Intn(5)+3) * time.Second
		}
		delays[i] = d
	}
	return delays
}
