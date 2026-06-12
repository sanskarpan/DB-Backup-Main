//go:build grpc

package interceptors

import (
	"context"
	"runtime/debug"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// contextKey is a custom type for context keys to avoid collisions
type contextKey string

const (
	requestIDKey contextKey = "request-id"
	userIDKey    contextKey = "user-id"
	userRoleKey  contextKey = "user-role"
)

// UnaryLoggingInterceptor logs all unary RPC calls
func UnaryLoggingInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		start := time.Now()

		// Get request ID from context
		requestID := getRequestID(ctx)

		log.Info().
			Str("request_id", requestID).
			Str("method", info.FullMethod).
			Msg("gRPC request started")

		// Call the handler
		resp, err := handler(ctx, req)

		// Log completion
		duration := time.Since(start)
		if err != nil {
			log.Error().
				Err(err).
				Str("request_id", requestID).
				Str("method", info.FullMethod).
				Dur("duration", duration).
				Msg("gRPC request failed")
		} else {
			log.Info().
				Str("request_id", requestID).
				Str("method", info.FullMethod).
				Dur("duration", duration).
				Msg("gRPC request completed")
		}

		return resp, err
	}
}

// StreamLoggingInterceptor logs all streaming RPC calls
func StreamLoggingInterceptor() grpc.StreamServerInterceptor {
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		start := time.Now()

		// Get request ID from context
		requestID := getRequestID(ss.Context())

		log.Info().
			Str("request_id", requestID).
			Str("method", info.FullMethod).
			Bool("is_client_stream", info.IsClientStream).
			Bool("is_server_stream", info.IsServerStream).
			Msg("gRPC stream started")

		// Call the handler
		err := handler(srv, ss)

		// Log completion
		duration := time.Since(start)
		if err != nil {
			log.Error().
				Err(err).
				Str("request_id", requestID).
				Str("method", info.FullMethod).
				Dur("duration", duration).
				Msg("gRPC stream failed")
		} else {
			log.Info().
				Str("request_id", requestID).
				Str("method", info.FullMethod).
				Dur("duration", duration).
				Msg("gRPC stream completed")
		}

		return err
	}
}

// UnaryAuthInterceptor performs authentication for unary RPCs
func UnaryAuthInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		// Skip auth for health check endpoints
		if info.FullMethod == "/grpc.health.v1.Health/Check" ||
			info.FullMethod == "/dbbackup.v1.MonitoringService/GetSystemHealth" {
			return handler(ctx, req)
		}

		// Extract metadata from context
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return nil, status.Error(codes.Unauthenticated, "missing metadata")
		}

		// Get authorization token
		tokens := md.Get("authorization")
		if len(tokens) == 0 {
			return nil, status.Error(codes.Unauthenticated, "missing authorization token")
		}

		token := tokens[0]

		// Validate JWT token
		userID, userRole, err := validateToken(token)
		if err != nil {
			return nil, status.Errorf(codes.Unauthenticated, "invalid token: %v", err)
		}

		// Add user info to context
		ctx = context.WithValue(ctx, userIDKey, userID)
		ctx = context.WithValue(ctx, userRoleKey, userRole)

		// Call the handler with authenticated context
		return handler(ctx, req)
	}
}

// StreamAuthInterceptor performs authentication for streaming RPCs
func StreamAuthInterceptor() grpc.StreamServerInterceptor {
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		// Skip auth for health check endpoints
		if info.FullMethod == "/grpc.health.v1.Health/Watch" {
			return handler(srv, ss)
		}

		// Extract metadata from context
		md, ok := metadata.FromIncomingContext(ss.Context())
		if !ok {
			return status.Error(codes.Unauthenticated, "missing metadata")
		}

		// Get authorization token
		tokens := md.Get("authorization")
		if len(tokens) == 0 {
			return status.Error(codes.Unauthenticated, "missing authorization token")
		}

		token := tokens[0]

		// Validate token
		userID, userRole, err := validateToken(token)
		if err != nil {
			return status.Errorf(codes.Unauthenticated, "invalid token: %v", err)
		}

		// Create wrapped stream with authenticated context
		ctx := context.WithValue(ss.Context(), userIDKey, userID)
		ctx = context.WithValue(ctx, userRoleKey, userRole)
		wrappedStream := &wrappedServerStream{ss, ctx}

		// Call the handler with authenticated stream
		return handler(srv, wrappedStream)
	}
}

// UnaryRecoveryInterceptor recovers from panics in unary RPCs
func UnaryRecoveryInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
		defer func() {
			if r := recover(); r != nil {
				requestID := getRequestID(ctx)

				log.Error().
					Str("request_id", requestID).
					Str("method", info.FullMethod).
					Interface("panic", r).
					Str("stack", string(debug.Stack())).
					Msg("gRPC panic recovered")

				err = status.Errorf(codes.Internal, "internal server error")
			}
		}()

		return handler(ctx, req)
	}
}

// StreamRecoveryInterceptor recovers from panics in streaming RPCs
func StreamRecoveryInterceptor() grpc.StreamServerInterceptor {
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) (err error) {
		defer func() {
			if r := recover(); r != nil {
				requestID := getRequestID(ss.Context())

				log.Error().
					Str("request_id", requestID).
					Str("method", info.FullMethod).
					Interface("panic", r).
					Str("stack", string(debug.Stack())).
					Msg("gRPC stream panic recovered")

				err = status.Errorf(codes.Internal, "internal server error")
			}
		}()

		return handler(srv, ss)
	}
}

// UnaryMetricsInterceptor collects metrics for unary RPCs
func UnaryMetricsInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		start := time.Now()

		// Call the handler
		resp, err := handler(ctx, req)

		// Record metrics
		duration := time.Since(start)
		recordUnaryMetrics(info.FullMethod, err, duration)

		return resp, err
	}
}

// StreamMetricsInterceptor collects metrics for streaming RPCs
func StreamMetricsInterceptor() grpc.StreamServerInterceptor {
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		start := time.Now()

		// Call the handler
		err := handler(srv, ss)

		// Record metrics
		duration := time.Since(start)
		recordStreamMetrics(info.FullMethod, err, duration, info.IsClientStream, info.IsServerStream)

		return err
	}
}

// UnaryRequestIDInterceptor adds a request ID to unary RPCs
func UnaryRequestIDInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		// Generate or extract request ID
		requestID := extractOrGenerateRequestID(ctx)

		// Add request ID to context
		ctx = context.WithValue(ctx, requestIDKey, requestID)

		// Add request ID to outgoing metadata
		ctx = metadata.AppendToOutgoingContext(ctx, "x-request-id", requestID)

		return handler(ctx, req)
	}
}

// StreamRequestIDInterceptor adds a request ID to streaming RPCs
func StreamRequestIDInterceptor() grpc.StreamServerInterceptor {
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		// Generate or extract request ID
		requestID := extractOrGenerateRequestID(ss.Context())

		// Create context with request ID
		ctx := context.WithValue(ss.Context(), requestIDKey, requestID)

		// Create wrapped stream with request ID
		wrappedStream := &wrappedServerStream{ss, ctx}

		return handler(srv, wrappedStream)
	}
}

// UnaryRateLimitInterceptor rate limits unary RPCs
func UnaryRateLimitInterceptor(limiter RateLimiter) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		// Get user ID from context
		userID := getUserID(ctx)

		// Check rate limit
		if !limiter.Allow(userID, info.FullMethod) {
			return nil, status.Error(codes.ResourceExhausted, "rate limit exceeded")
		}

		return handler(ctx, req)
	}
}

// StreamRateLimitInterceptor rate limits streaming RPCs
func StreamRateLimitInterceptor(limiter RateLimiter) grpc.StreamServerInterceptor {
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		// Get user ID from context
		userID := getUserID(ss.Context())

		// Check rate limit
		if !limiter.Allow(userID, info.FullMethod) {
			return status.Error(codes.ResourceExhausted, "rate limit exceeded")
		}

		return handler(srv, ss)
	}
}

// Helper types and functions

// wrappedServerStream wraps a grpc.ServerStream with a custom context
type wrappedServerStream struct {
	grpc.ServerStream
	ctx context.Context
}

// Context returns the custom context
func (w *wrappedServerStream) Context() context.Context {
	return w.ctx
}

// RateLimiter interface for rate limiting
type RateLimiter interface {
	Allow(userID, method string) bool
}

// SimpleRateLimiter is a sliding window rate limiter
type SimpleRateLimiter struct {
	requestsPerMinute int
	window            time.Duration
	requests          map[string][]time.Time
	mu                sync.RWMutex
	cleanupTicker     *time.Ticker
	stopCleanup       chan bool
}

// NewSimpleRateLimiter creates a new rate limiter with sliding window
func NewSimpleRateLimiter(requestsPerMinute int) *SimpleRateLimiter {
	limiter := &SimpleRateLimiter{
		requestsPerMinute: requestsPerMinute,
		window:            time.Minute,
		requests:          make(map[string][]time.Time),
		cleanupTicker:     time.NewTicker(time.Minute),
		stopCleanup:       make(chan bool),
	}

	// Start cleanup goroutine
	go limiter.cleanup()

	return limiter
}

// Allow checks if a request should be allowed using sliding window
func (l *SimpleRateLimiter) Allow(userID, method string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	key := userID + ":" + method

	// Get or create request history for this key
	requests, exists := l.requests[key]
	if !exists {
		requests = make([]time.Time, 0)
	}

	// Remove requests outside the window
	cutoff := now.Add(-l.window)
	validRequests := make([]time.Time, 0)
	for _, reqTime := range requests {
		if reqTime.After(cutoff) {
			validRequests = append(validRequests, reqTime)
		}
	}

	// Check if limit is exceeded
	if len(validRequests) >= l.requestsPerMinute {
		l.requests[key] = validRequests
		return false
	}

	// Add current request
	validRequests = append(validRequests, now)
	l.requests[key] = validRequests

	return true
}

// cleanup removes old entries periodically
func (l *SimpleRateLimiter) cleanup() {
	for {
		select {
		case <-l.cleanupTicker.C:
			l.mu.Lock()
			now := time.Now()
			cutoff := now.Add(-l.window)

			for key, requests := range l.requests {
				validRequests := make([]time.Time, 0)
				for _, reqTime := range requests {
					if reqTime.After(cutoff) {
						validRequests = append(validRequests, reqTime)
					}
				}

				if len(validRequests) == 0 {
					delete(l.requests, key)
				} else {
					l.requests[key] = validRequests
				}
			}
			l.mu.Unlock()

		case <-l.stopCleanup:
			return
		}
	}
}

// Stop stops the rate limiter cleanup goroutine
func (l *SimpleRateLimiter) Stop() {
	l.cleanupTicker.Stop()
	l.stopCleanup <- true
}

// Helper functions

func extractOrGenerateRequestID(ctx context.Context) string {
	md, ok := metadata.FromIncomingContext(ctx)
	if ok {
		if ids := md.Get("x-request-id"); len(ids) > 0 {
			return ids[0]
		}
	}
	return uuid.New().String()
}

func getRequestID(ctx context.Context) string {
	if id, ok := ctx.Value(requestIDKey).(string); ok {
		return id
	}
	return "unknown"
}

func getUserID(ctx context.Context) string {
	if id, ok := ctx.Value(userIDKey).(string); ok {
		return id
	}
	return "anonymous"
}

func validateToken(token string) (userID, userRole string, err error) {
	// Use the default JWT validator
	validator := GetDefaultValidator()
	return validator.ValidateToken(token)
}

// MetricsRecorder interface for recording metrics
type MetricsRecorder interface {
	RecordUnaryRPC(method string, statusCode codes.Code, duration time.Duration)
	RecordStreamRPC(method string, statusCode codes.Code, duration time.Duration, streamType string)
}

// DefaultMetricsRecorder is a simple in-memory metrics recorder
type DefaultMetricsRecorder struct {
	unaryMetrics  map[string]*RPCMetrics
	streamMetrics map[string]*RPCMetrics
	mu            sync.RWMutex
}

// RPCMetrics holds metrics for a specific RPC method
type RPCMetrics struct {
	TotalRequests   int64
	SuccessRequests int64
	FailedRequests  int64
	TotalDuration   time.Duration
	MinDuration     time.Duration
	MaxDuration     time.Duration
	LastUpdated     time.Time
}

var defaultRecorder = NewDefaultMetricsRecorder()

// NewDefaultMetricsRecorder creates a new metrics recorder
func NewDefaultMetricsRecorder() *DefaultMetricsRecorder {
	return &DefaultMetricsRecorder{
		unaryMetrics:  make(map[string]*RPCMetrics),
		streamMetrics: make(map[string]*RPCMetrics),
	}
}

// RecordUnaryRPC records metrics for a unary RPC
func (r *DefaultMetricsRecorder) RecordUnaryRPC(method string, statusCode codes.Code, duration time.Duration) {
	r.mu.Lock()
	defer r.mu.Unlock()

	metrics, exists := r.unaryMetrics[method]
	if !exists {
		metrics = &RPCMetrics{
			MinDuration: duration,
			MaxDuration: duration,
		}
		r.unaryMetrics[method] = metrics
	}

	metrics.TotalRequests++
	if statusCode == codes.OK {
		metrics.SuccessRequests++
	} else {
		metrics.FailedRequests++
	}

	metrics.TotalDuration += duration
	if duration < metrics.MinDuration {
		metrics.MinDuration = duration
	}
	if duration > metrics.MaxDuration {
		metrics.MaxDuration = duration
	}
	metrics.LastUpdated = time.Now()
}

// RecordStreamRPC records metrics for a streaming RPC
func (r *DefaultMetricsRecorder) RecordStreamRPC(method string, statusCode codes.Code, duration time.Duration, streamType string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	key := method + ":" + streamType
	metrics, exists := r.streamMetrics[key]
	if !exists {
		metrics = &RPCMetrics{
			MinDuration: duration,
			MaxDuration: duration,
		}
		r.streamMetrics[key] = metrics
	}

	metrics.TotalRequests++
	if statusCode == codes.OK {
		metrics.SuccessRequests++
	} else {
		metrics.FailedRequests++
	}

	metrics.TotalDuration += duration
	if duration < metrics.MinDuration {
		metrics.MinDuration = duration
	}
	if duration > metrics.MaxDuration {
		metrics.MaxDuration = duration
	}
	metrics.LastUpdated = time.Now()
}

// GetUnaryMetrics returns all unary RPC metrics
func (r *DefaultMetricsRecorder) GetUnaryMetrics() map[string]*RPCMetrics {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make(map[string]*RPCMetrics)
	for k, v := range r.unaryMetrics {
		result[k] = v
	}
	return result
}

// GetStreamMetrics returns all stream RPC metrics
func (r *DefaultMetricsRecorder) GetStreamMetrics() map[string]*RPCMetrics {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make(map[string]*RPCMetrics)
	for k, v := range r.streamMetrics {
		result[k] = v
	}
	return result
}

// SetDefaultMetricsRecorder sets the global default metrics recorder
func SetDefaultMetricsRecorder(recorder *DefaultMetricsRecorder) {
	defaultRecorder = recorder
}

// GetDefaultMetricsRecorder returns the global default metrics recorder
func GetDefaultMetricsRecorder() *DefaultMetricsRecorder {
	return defaultRecorder
}

func recordUnaryMetrics(method string, err error, duration time.Duration) {
	statusCode := codes.OK
	if err != nil {
		st, ok := status.FromError(err)
		if ok {
			statusCode = st.Code()
		}
	}

	// Record to metrics recorder
	defaultRecorder.RecordUnaryRPC(method, statusCode, duration)

	log.Debug().
		Str("method", method).
		Str("status", statusCode.String()).
		Dur("duration", duration).
		Msg("gRPC unary metrics")
}

func recordStreamMetrics(method string, err error, duration time.Duration, isClientStream, isServerStream bool) {
	statusCode := codes.OK
	if err != nil {
		st, ok := status.FromError(err)
		if ok {
			statusCode = st.Code()
		}
	}

	streamType := "unknown"
	if isClientStream && isServerStream {
		streamType = "bidi"
	} else if isClientStream {
		streamType = "client"
	} else if isServerStream {
		streamType = "server"
	}

	// Record to metrics recorder
	defaultRecorder.RecordStreamRPC(method, statusCode, duration, streamType)

	log.Debug().
		Str("method", method).
		Str("status", statusCode.String()).
		Str("stream_type", streamType).
		Dur("duration", duration).
		Msg("gRPC stream metrics")
}
