package api

import (
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"sy-vpn-backend/internal/auth"
)

// RateLimitRegister wraps a handler with a per-IP token bucket, aimed at
// POST /auth/register specifically: that route is deliberately unauthenticated
// (see handleRegister's doc comment — the app registers itself with no login
// screen), which means it's just as reachable by a script making up a fresh
// random device_id on every call as by a real install. A real device only
// ever calls this once in its lifetime (the ID is persisted locally and
// reused), so even a household or office sharing one public IP registers a
// handful of real devices over time, never dozens within minutes — a bot
// looks completely different from that.
//
// Deliberately generous (see burst/refill below): many real users in Myanmar
// sit behind carrier-grade NAT or a shared home/office WiFi, so many
// unrelated real installs can legitimately share one IP. The limit exists to
// stop a script hammering the endpoint continuously, not to police normal
// shared-IP traffic.
func RateLimitRegister(next http.HandlerFunc) http.HandlerFunc {
	const (
		burst          = 20              // allows a fair number of installs from one shared IP in a short window
		refillInterval = 3 * time.Minute // steady-state: 20 per hour per IP
		bucketTTL      = 2 * time.Hour   // idle buckets are forgotten so the map doesn't grow forever
	)

	var (
		mu      sync.Mutex
		buckets = map[string]*ipBucket{}
	)

	// Evicts idle entries periodically so long-running deployments don't
	// slowly accumulate one bucket per IP ever seen.
	go func() {
		for range time.Tick(30 * time.Minute) {
			mu.Lock()
			now := time.Now()
			for ip, b := range buckets {
				if now.Sub(b.lastSeen) > bucketTTL {
					delete(buckets, ip)
				}
			}
			mu.Unlock()
		}
	}()

	return func(w http.ResponseWriter, r *http.Request) {
		ip := clientIP(r)

		mu.Lock()
		b, ok := buckets[ip]
		if !ok {
			b = &ipBucket{tokens: burst, lastSeen: time.Now()}
			buckets[ip] = b
		}
		allowed := b.take(burst, refillInterval)
		mu.Unlock()

		if !allowed {
			http.Error(w, "too many registration attempts, try again later", http.StatusTooManyRequests)
			return
		}
		next(w, r)
	}
}

type ipBucket struct {
	tokens   float64
	lastSeen time.Time
}

// take refills the bucket for elapsed time (one token per refillInterval, up
// to burst), then consumes a token if one is available.
func (b *ipBucket) take(burst int, refillInterval time.Duration) bool {
	now := time.Now()
	elapsed := now.Sub(b.lastSeen)
	b.lastSeen = now

	b.tokens += elapsed.Seconds() / refillInterval.Seconds()
	if b.tokens > float64(burst) {
		b.tokens = float64(burst)
	}

	if b.tokens < 1 {
		return false
	}
	b.tokens--
	return true
}

// userLimiter is a per-authenticated-user token bucket (same bucket logic as
// RateLimitRegister). Must be wrapped INSIDE auth.Require so the user is known.
type userLimiter struct {
	mu             sync.Mutex
	buckets        map[string]*ipBucket
	burst          int
	refillInterval time.Duration
	lastSweep      time.Time
}

func newUserLimiter(burst int, refillInterval time.Duration) *userLimiter {
	return &userLimiter{buckets: map[string]*ipBucket{}, burst: burst, refillInterval: refillInterval, lastSweep: time.Now()}
}

func (l *userLimiter) allow(userID string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := time.Now()
	// Opportunistic eviction of idle buckets so the map cannot grow forever.
	if now.Sub(l.lastSweep) > 30*time.Minute {
		for k, b := range l.buckets {
			if now.Sub(b.lastSeen) > time.Hour {
				delete(l.buckets, k)
			}
		}
		l.lastSweep = now
	}
	b, ok := l.buckets[userID]
	if !ok {
		b = &ipBucket{tokens: float64(l.burst), lastSeen: now}
		l.buckets[userID] = b
	}
	return b.take(l.burst, l.refillInterval)
}

func (l *userLimiter) wrap(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, ok := auth.UserFromContext(r.Context())
		if ok && !l.allow(user.ID) {
			w.Header().Set("Retry-After", "10")
			http.Error(w, "too many requests, try again shortly", http.StatusTooManyRequests)
			return
		}
		next(w, r)
	}
}

// isTrustedProxy reports whether the direct TCP peer is loopback or a private
// address, i.e. cloudflared / a local reverse proxy rather than the internet.
func isTrustedProxy(remoteAddr string) bool {
	host, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		host = remoteAddr
	}
	ip := net.ParseIP(host)
	return ip != nil && (ip.IsLoopback() || ip.IsPrivate())
}

// clientIP resolves the real visitor IP behind Cloudflare Tunnel: this
// server binds to 127.0.0.1 and is only reached via cloudflared, which sets
// Cf-Connecting-Ip. Those headers are only believed when the direct peer is
// loopback/private; otherwise anyone could spoof them to dodge the limiter,
// so the real RemoteAddr (host part) is used.
func clientIP(r *http.Request) string {
	if isTrustedProxy(r.RemoteAddr) {
		if ip := strings.TrimSpace(r.Header.Get("Cf-Connecting-Ip")); ip != "" {
			return ip
		}
		if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
			parts := strings.Split(xff, ",")
			return strings.TrimSpace(parts[0])
		}
	}
	if host, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		return host
	}
	return r.RemoteAddr
}
