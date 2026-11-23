package utils

import (
	"context"
	"sync"
	"time"
)

// RateLimiter implementa un token bucket de ritmo constante.
//
// Los tokens se recargan a un ritmo de requests/perSeconds tokens por segundo.
// Wait reserva un token bajo el mutex y duerme fuera de él, de modo que N
// goroutines concurrentes no pueden despertar y pasar todas a la vez.
type RateLimiter struct {
	mu     sync.Mutex
	tokens float64
	burst  float64
	rate   float64 // tokens por segundo
	last   time.Time
}

// NewRateLimiter crea un limitador que permite requests peticiones cada
// perSeconds segundos. La primera petición pasa sin espera y el resto se
// espacia según el ritmo configurado.
func NewRateLimiter(requests int, perSeconds int) *RateLimiter {
	if requests <= 0 {
		requests = 45
	}
	if perSeconds <= 0 {
		perSeconds = 60
	}
	return &RateLimiter{
		tokens: 1,
		burst:  float64(requests),
		rate:   float64(requests) / float64(perSeconds),
		last:   time.Now(),
	}
}

// Wait bloquea hasta que haya un token disponible o el contexto se cancele.
func (rl *RateLimiter) Wait(ctx context.Context) error {
	if rl == nil {
		return nil
	}
	if ctx == nil {
		ctx = context.Background()
	}

	for {
		rl.mu.Lock()
		now := time.Now()
		rl.refillLocked(now)

		if rl.tokens >= 1 {
			rl.tokens--
			rl.mu.Unlock()
			return nil
		}

		// Calcular la espera sin retener el mutex: otras goroutines deben poder
		// reservar el token que quede libre.
		wait := time.Duration((1-rl.tokens)/rl.rate*float64(time.Second)) + time.Millisecond
		rl.mu.Unlock()

		timer := time.NewTimer(wait)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
	}
}

// refillLocked recarga los tokens acumulados desde la última llamada.
// Debe invocarse con rl.mu tomado.
func (rl *RateLimiter) refillLocked(now time.Time) {
	elapsed := now.Sub(rl.last).Seconds()
	if elapsed <= 0 {
		return
	}
	rl.tokens += elapsed * rl.rate
	if rl.tokens > rl.burst {
		rl.tokens = rl.burst
	}
	rl.last = now
}
