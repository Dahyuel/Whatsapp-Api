package antiban

import (
	"math/rand"
	"time"
)

// AdaptiveTiming adjusts message delays based on contextual signals.
type AdaptiveTiming struct {
	BaseDelay  time.Duration
	JitterMin  time.Duration
	JitterMax  time.Duration
}

// NewAdaptiveTiming creates an AdaptiveTiming engine with config values.
func NewAdaptiveTiming(base, jMin, jMax time.Duration) *AdaptiveTiming {
	return &AdaptiveTiming{
		BaseDelay: base,
		JitterMin: jMin,
		JitterMax: jMax,
	}
}

// ComputeDelay returns an adjusted delay based on:
//   - time of day (slower at night)
//   - message length (longer = more delay)
//   - recent conversation activity (active chat = faster)
//   - contact familiarity (new contact = slower)
func (a *AdaptiveTiming) ComputeDelay(msgLength int, hourOfDay int, isActiveChat bool, isNewContact bool) time.Duration {
	base := a.BaseDelay

	// Night-time penalty: 22:00–07:00 local → add 2–4s
	if hourOfDay >= 22 || hourOfDay < 7 {
		base += time.Duration(2+rand.Intn(3)) * time.Second
	}

	// Long messages → more delay to simulate more "thinking"
	if msgLength > 200 {
		base += 2 * time.Second
	} else if msgLength > 80 {
		base += 1 * time.Second
	}

	// Active chat shortcut: halve the delay
	if isActiveChat {
		base = base / 2
	}

	// New contact extra caution: add 3–6s extra
	if isNewContact {
		base += time.Duration(3+rand.Intn(4)) * time.Second
	}

	return JitterDelay(base, a.JitterMin, a.JitterMax)
}
