package middleware

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"net/http"
	"runtime/debug"
	"sync"
	"time"
)

type contextKey string

const RequestIDKey contextKey = "requestID"

// responseWriterWrapper wraps http.ResponseWriter to capture status code
type responseWriterWrapper struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriterWrapper) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// RequestID attaches a unique request identifier to incoming HTTP requests
func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reqID := r.Header.Get("X-Request-ID")
		if reqID == "" {
			reqID = generateID()
		}

		ctx := context.WithValue(r.Context(), RequestIDKey, reqID)
		w.Header().Set("X-Request-ID", reqID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func generateID() string {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(b)
}

// GetRequestID retrieves request ID from context
func GetRequestID(ctx context.Context) string {
	if reqID, ok := ctx.Value(RequestIDKey).(string); ok {
		return reqID
	}
	return ""
}

// Logger logs incoming requests and metrics
func Logger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		wrapper := &responseWriterWrapper{ResponseWriter: w, statusCode: http.StatusOK}

		next.ServeHTTP(wrapper, r)

		duration := time.Since(start)
		reqID := GetRequestID(r.Context())

		log.Printf("[%s] %s %s | Status: %d | Duration: %v | IP: %s | ReqID: %s",
			r.Method, r.URL.Path, r.Proto, wrapper.statusCode, duration, r.RemoteAddr, reqID)
	})
}

// RateLimiter manages IP-based token-bucket request throttling
type clientLimiter struct {
	tokens     float64
	lastUpdate time.Time
}

type IPRateLimiter struct {
	mu       sync.Mutex
	clients  map[string]*clientLimiter
	rate     float64 // tokens per second
	capacity float64
}

func NewIPRateLimiter(rps int, burst int) *IPRateLimiter {
	limiter := &IPRateLimiter{
		clients:  make(map[string]*clientLimiter),
		rate:     float64(rps),
		capacity: float64(burst),
	}

	// Cleanup stale IPs every 5 minutes
	go func() {
		for {
			time.Sleep(5 * time.Minute)
			limiter.mu.Lock()
			for ip, client := range limiter.clients {
				if time.Since(client.lastUpdate) > 10*time.Minute {
					delete(limiter.clients, ip)
				}
			}
			limiter.mu.Unlock()
		}
	}()

	return limiter
}

func (l *IPRateLimiter) Allow(ip string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	client, exists := l.clients[ip]
	if !exists {
		l.clients[ip] = &clientLimiter{
			tokens:     l.capacity - 1.0,
			lastUpdate: now,
		}
		return true
	}

	elapsed := now.Sub(client.lastUpdate).Seconds()
	client.tokens += elapsed * l.rate
	if client.tokens > l.capacity {
		client.tokens = l.capacity
	}
	client.lastUpdate = now

	if client.tokens >= 1.0 {
		client.tokens -= 1.0
		return true
	}

	return false
}

func RateLimit(limiter *IPRateLimiter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := r.RemoteAddr
			if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
				ip = forwarded
			}

			if !limiter.Allow(ip) {
				w.Header().Set("Content-Type", "application/json")
				w.Header().Set("Retry-After", "1")
				w.WriteHeader(http.StatusTooManyRequests)
				w.Write([]byte(`{"success":false,"error":"Rate limit exceeded. Please try again shortly."}`))
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// CORS adds Cross-Origin Resource Sharing headers for Web clients
func CORS(allowedOrigins string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			if origin == "" || allowedOrigins == "*" {
				w.Header().Set("Access-Control-Allow-Origin", "*")
			} else {
				w.Header().Set("Access-Control-Allow-Origin", allowedOrigins)
			}

			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With, X-Request-ID, X-Secret-App-Key, X-App-Secret-Key")
			w.Header().Set("Access-Control-Expose-Headers", "X-Request-ID")
			w.Header().Set("Access-Control-Max-Age", "86400")

			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// Recovery handles panics cleanly without crashing the server
func Recovery(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				reqID := GetRequestID(r.Context())
				log.Printf("[PANIC RECOVERY] ReqID: %s | Error: %v\nStack: %s", reqID, err, string(debug.Stack()))
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte(fmt.Sprintf(`{"success":false,"error":"Internal Server Error","request_id":"%s"}`, reqID)))
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// SecurityHeaders injects standard web security headers
func SecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("X-XSS-Protection", "1; mode=block")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		next.ServeHTTP(w, r)
	})
}
