package utils

import (
	"sync"
	"time"
)

type RateLimiter struct {
	mu       sync.Mutex
	tokens   float64
	max      float64
	interval time.Duration
	last     time.Time
}

func NewRateLimiter(requests int, perSeconds int) *RateLimiter {
	if requests <= 0 {
		requests = 45
	}
	if perSeconds <= 0 {
		perSeconds = 60
	}
	return &RateLimiter{
		tokens:   float64(requests),
		max:      float64(requests),
		interval: time.Duration(perSeconds) * time.Second / time.Duration(requests),
		last:     time.Now(),
	}
}

func (rl *RateLimiter) Wait() {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(rl.last)
	rl.last = now

	rl.tokens += elapsed.Seconds() * (rl.max / rl.interval.Seconds())
	if rl.tokens > rl.max {
		rl.tokens = rl.max
	}

	if rl.tokens < 1 {
		sleepDuration := time.Duration((1 - rl.tokens) * rl.interval.Seconds() * float64(time.Second))
		rl.mu.Unlock()
		time.Sleep(sleepDuration)
		rl.mu.Lock()
		rl.tokens = 0
	} else {
		rl.tokens--
	}
}
