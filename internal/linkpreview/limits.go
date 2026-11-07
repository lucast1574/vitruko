package linkpreview

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

// SourceRateLimit is a store restriction, distinct from the API user's quota.
type SourceRateLimit struct{ Until time.Time }

func (e *SourceRateLimit) Error() string {
	return "La tienda limita temporalmente las consultas. Respeta Retry-After y vuelve a intentarlo después."
}
func (e *SourceRateLimit) RetryAfter() int { return max(1, int(time.Until(e.Until).Seconds())+1) }
func sourceLimit(header string) *SourceRateLimit {
	now := time.Now()
	until := now.Add(time.Minute)
	if seconds, err := strconv.Atoi(header); err == nil && seconds >= 0 {
		until = now.Add(time.Duration(seconds) * time.Second)
	} else if date, err := http.ParseTime(header); err == nil && date.After(now) {
		until = date
	}
	return &SourceRateLimit{Until: until}
}

type hostGate struct {
	slot                   chan struct{}
	next, blocked, touched time.Time
}
type sourceScheduler struct {
	mu       sync.Mutex
	hosts    map[string]*hostGate
	interval time.Duration
}

var sharedSources = &sourceScheduler{hosts: map[string]*hostGate{}, interval: 500 * time.Millisecond}

func (s *sourceScheduler) acquire(ctx context.Context, host string) (func(), error) {
	s.mu.Lock()
	gate := s.hosts[host]
	if gate == nil {
		if len(s.hosts) >= 512 {
			for name, g := range s.hosts {
				if len(g.slot) == 0 && time.Since(g.touched) > 10*time.Minute && time.Now().After(g.blocked) {
					delete(s.hosts, name)
				}
			}
		}
		if len(s.hosts) >= 512 {
			s.mu.Unlock()
			return nil, errors.New("too many active store destinations")
		}
		gate = &hostGate{slot: make(chan struct{}, 1)}
		s.hosts[host] = gate
	}
	gate.touched = time.Now()
	blocked := gate.blocked
	s.mu.Unlock()
	if time.Now().Before(blocked) {
		return nil, &SourceRateLimit{Until: blocked}
	}
	select {
	case gate.slot <- struct{}{}:
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	release := func() { <-gate.slot }
	s.mu.Lock()
	blocked, wait := gate.blocked, time.Until(gate.next)
	s.mu.Unlock()
	if time.Now().Before(blocked) {
		release()
		return nil, &SourceRateLimit{Until: blocked}
	}
	if wait > 0 {
		timer := time.NewTimer(wait)
		defer timer.Stop()
		select {
		case <-timer.C:
		case <-ctx.Done():
			release()
			return nil, ctx.Err()
		}
	}
	s.mu.Lock()
	gate.next = time.Now().Add(s.interval)
	s.mu.Unlock()
	return release, nil
}

type politeTransport struct {
	base      http.RoundTripper
	scheduler *sourceScheduler
}

func (t *politeTransport) CloseIdleConnections() {
	if closer, ok := t.base.(interface{ CloseIdleConnections() }); ok {
		closer.CloseIdleConnections()
	}
}

func (t *politeTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	host := strings.ToLower(strings.TrimSuffix(req.URL.Hostname(), "."))
	release, err := t.scheduler.acquire(req.Context(), host)
	if err != nil {
		return nil, err
	}
	defer release()
	res, err := t.base.RoundTrip(req)
	if err == nil && res.StatusCode == http.StatusTooManyRequests {
		limited := sourceLimit(res.Header.Get("Retry-After"))
		t.scheduler.mu.Lock()
		g := t.scheduler.hosts[host]
		if limited.Until.After(g.blocked) {
			g.blocked = limited.Until
		}
		t.scheduler.mu.Unlock()
		res.Body.Close()
		return nil, limited
	}
	return res, err
}
