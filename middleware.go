package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Logging middleware
type LogEntry struct {
	Timestamp       time.Time
	Method          string
	Path            string
	SourceIP        string
	StatusCode      int
	ResponseTime    time.Duration
	RequestSize     int64
	ResponseSize    int
	HeadersReceived string
	AuthSuccess     bool
}

var logFile *os.File

func initLogger() {
	var err error
	logFile, err = os.OpenFile("logs/api_requests.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatal(err)
	}
}

func logRequest(entry LogEntry) {
	logLine := fmt.Sprintf("[%s] %s %s | IP: %s | Status: %d | Duration: %dms | ReqSize: %d bytes | RespSize: %d bytes | Headers: %s | Auth: %t\n",
		entry.Timestamp.Format("2006-01-02 15:04:05"),
		entry.Method,
		entry.Path,
		entry.SourceIP,
		entry.StatusCode,
		entry.ResponseTime.Milliseconds(),
		entry.RequestSize,
		entry.ResponseSize,
		entry.HeadersReceived,
		entry.AuthSuccess,
	)
	logFile.WriteString(logLine)
	log.Print(logLine)
}

type responseWriter struct {
	http.ResponseWriter
	statusCode      int
	size            int
	isAuthenticated bool
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	size, err := rw.ResponseWriter.Write(b)
	rw.size += size
	return size, err
}

func loggingMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		rw := &responseWriter{
			ResponseWriter:  w,
			statusCode:      200,
			isAuthenticated: true,
		}

		// Get headers
		headers := []string{}
		for key := range r.Header {
			headers = append(headers, key)
		}

		next(rw, r)

		duration := time.Since(start)
		entry := LogEntry{
			Timestamp:       start,
			Method:          r.Method,
			Path:            r.URL.Path,
			SourceIP:        r.RemoteAddr,
			StatusCode:      rw.statusCode,
			ResponseTime:    duration,
			RequestSize:     r.ContentLength,
			ResponseSize:    rw.size,
			HeadersReceived: strings.Join(headers, ", "),
			AuthSuccess:     rw.isAuthenticated,
		}

		logRequest(entry)

	}
}

func (a *App) routingMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		next.ServeHTTP(w, r.WithContext(a.Context))
	}
}

// Rate Limiting Middleware
func (a *App) rateLimitMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		studentNo := r.Context().Value("LOGGEDIN_STUDENT_NO").(string)
		dailyLimit, err := a.Queries.GetStudentDailyLimit(r.Context(), studentNo)

		if err != nil {
			http.Error(w, "Gateway Error", http.StatusBadGateway)
			return
		}

		if dailyLimit == 0 {
			http.Error(w, "You've reached your daily limit.", http.StatusBadRequest)
			return
		}
		next.ServeHTTP(w, r)

		a.Queries.DecreasePaymentLimit(r.Context(), studentNo)
	}
}

// Authentication middleware

func authMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tokenStr := extractToken(r)

		if tokenStr == "" {
			http.Error(w, "Missing or invalid Authorization header", http.StatusUnauthorized)
			return
		}

		claims := &jwt.RegisteredClaims{}
		token, err := jwt.ParseWithClaims(tokenStr, claims, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, jwt.ErrSignatureInvalid
			}

			secret := os.Getenv("JWT_SECRET")
			if secret == "" {
				return nil, fmt.Errorf("JWT_SECRET is empty")
			}

			return []byte(secret), nil
		})
		if err != nil || !token.Valid {
			msg := fmt.Errorf("Token Status: %w", err)
			// http.Error(w, msg.Error(), http.StatusUnauthorized)
			http.Error(w, "Invalid token"+msg.Error(), http.StatusUnauthorized)
			return
		}
		// Attach user ID (sub) to context for handlers
		ctx := context.WithValue(r.Context(), "LOGGEDIN_STUDENT_NO", claims.Subject)
		next.ServeHTTP(w, r.WithContext(ctx))
	}
}
