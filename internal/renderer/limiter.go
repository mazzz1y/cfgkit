package renderer

import (
	"context"
	"errors"
	"time"
)

var ErrCheckBusy = errors.New("check: no free slot")

type Limiter struct {
	slots chan struct{}
}

func NewLimiter(n int) *Limiter {
	if n < 1 {
		n = 1
	}
	return &Limiter{slots: make(chan struct{}, n)}
}

func (l *Limiter) acquire(ctx context.Context, wait time.Duration) (func(), error) {
	if l == nil {
		return func() {}, nil
	}

	timer := time.NewTimer(wait)
	defer timer.Stop()

	select {
	case l.slots <- struct{}{}:
		return func() { <-l.slots }, nil
	case <-timer.C:
		return nil, ErrCheckBusy
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}
