//go:build grpc

package interceptors

import (
	"context"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// Mock handler for testing
func mockUnaryHandler(ctx context.Context, req interface{}) (interface{}, error) {
	return "success", nil
}

func mockErrorHandler(ctx context.Context, req interface{}) (interface{}, error) {
	return nil, status.Error(codes.Internal, "test error")
}

func mockPanicHandler(ctx context.Context, req interface{}) (interface{}, error) {
	panic("test panic")
}

// Mock server stream for testing
type mockServerStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (m *mockServerStream) Context() context.Context {
	return m.ctx
}

func TestUnaryRequestIDInterceptor(t *testing.T) {
	interceptor := UnaryRequestIDInterceptor()

	ctx := context.Background()
	info := &grpc.UnaryServerInfo{
		FullMethod: "/test.Service/Method",
	}

	resp, err := interceptor(ctx, nil, info, mockUnaryHandler)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if resp != "success" {
		t.Errorf("Expected 'success', got %v", resp)
	}

	t.Log("✓ Request ID interceptor works correctly")
}

func TestUnaryRequestIDInterceptorWithExistingID(t *testing.T) {
	interceptor := UnaryRequestIDInterceptor()

	// Create context with existing request ID
	md := metadata.New(map[string]string{
		"x-request-id": "existing-id-123",
	})
	ctx := metadata.NewIncomingContext(context.Background(), md)

	info := &grpc.UnaryServerInfo{
		FullMethod: "/test.Service/Method",
	}

	// Capture the request ID from context
	var capturedID string
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		capturedID = getRequestID(ctx)
		return "success", nil
	}

	_, err := interceptor(ctx, nil, info, handler)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if capturedID != "existing-id-123" {
		t.Errorf("Expected request ID 'existing-id-123', got %s", capturedID)
	}

	t.Log("✓ Request ID extraction from metadata works correctly")
}

func TestUnaryAuthInterceptor(t *testing.T) {
	// First set up a JWT validator with a known secret
	validator := NewJWTValidator("test-secret", "test-issuer")
	SetDefaultValidator(validator)

	// Create a valid token
	token, err := validator.CreateToken("user-123", "admin", 1*time.Hour)
	if err != nil {
		t.Fatalf("Failed to create token: %v", err)
	}

	interceptor := UnaryAuthInterceptor()

	// Create context with authorization token
	md := metadata.New(map[string]string{
		"authorization": "Bearer " + token,
	})
	ctx := metadata.NewIncomingContext(context.Background(), md)

	info := &grpc.UnaryServerInfo{
		FullMethod: "/test.Service/Method",
	}

	// Capture user info from context
	var capturedUserID, capturedUserRole string
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		if id, ok := ctx.Value(userIDKey).(string); ok {
			capturedUserID = id
		}
		if role, ok := ctx.Value(userRoleKey).(string); ok {
			capturedUserRole = role
		}
		return "success", nil
	}

	resp, err := interceptor(ctx, nil, info, handler)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if resp != "success" {
		t.Errorf("Expected 'success', got %v", resp)
	}

	if capturedUserID != "user-123" {
		t.Errorf("Expected user ID 'user-123', got %s", capturedUserID)
	}

	if capturedUserRole != "admin" {
		t.Errorf("Expected user role 'admin', got %s", capturedUserRole)
	}

	t.Log("✓ Auth interceptor with valid token works correctly")
}

func TestUnaryAuthInterceptorMissingToken(t *testing.T) {
	interceptor := UnaryAuthInterceptor()

	ctx := context.Background()
	info := &grpc.UnaryServerInfo{
		FullMethod: "/test.Service/Method",
	}

	_, err := interceptor(ctx, nil, info, mockUnaryHandler)

	if err == nil {
		t.Fatal("Expected error for missing metadata")
	}

	st, ok := status.FromError(err)
	if !ok {
		t.Fatal("Expected gRPC status error")
	}

	if st.Code() != codes.Unauthenticated {
		t.Errorf("Expected Unauthenticated code, got %v", st.Code())
	}

	t.Log("✓ Auth interceptor rejects missing token correctly")
}

func TestUnaryAuthInterceptorInvalidToken(t *testing.T) {
	interceptor := UnaryAuthInterceptor()

	// Create context with invalid token
	md := metadata.New(map[string]string{
		"authorization": "Bearer invalid-token",
	})
	ctx := metadata.NewIncomingContext(context.Background(), md)

	info := &grpc.UnaryServerInfo{
		FullMethod: "/test.Service/Method",
	}

	_, err := interceptor(ctx, nil, info, mockUnaryHandler)

	if err == nil {
		t.Fatal("Expected error for invalid token")
	}

	st, ok := status.FromError(err)
	if !ok {
		t.Fatal("Expected gRPC status error")
	}

	if st.Code() != codes.Unauthenticated {
		t.Errorf("Expected Unauthenticated code, got %v", st.Code())
	}

	t.Log("✓ Auth interceptor rejects invalid token correctly")
}

func TestUnaryAuthInterceptorHealthCheckSkip(t *testing.T) {
	interceptor := UnaryAuthInterceptor()

	ctx := context.Background()
	info := &grpc.UnaryServerInfo{
		FullMethod: "/grpc.health.v1.Health/Check",
	}

	resp, err := interceptor(ctx, nil, info, mockUnaryHandler)

	if err != nil {
		t.Fatalf("Expected no error for health check, got %v", err)
	}

	if resp != "success" {
		t.Errorf("Expected 'success', got %v", resp)
	}

	t.Log("✓ Auth interceptor skips health checks correctly")
}

func TestUnaryRecoveryInterceptor(t *testing.T) {
	interceptor := UnaryRecoveryInterceptor()

	ctx := context.Background()
	info := &grpc.UnaryServerInfo{
		FullMethod: "/test.Service/Method",
	}

	_, err := interceptor(ctx, nil, info, mockPanicHandler)

	if err == nil {
		t.Fatal("Expected error from panic recovery")
	}

	st, ok := status.FromError(err)
	if !ok {
		t.Fatal("Expected gRPC status error")
	}

	if st.Code() != codes.Internal {
		t.Errorf("Expected Internal code, got %v", st.Code())
	}

	t.Log("✓ Recovery interceptor handles panics correctly")
}

func TestUnaryLoggingInterceptor(t *testing.T) {
	interceptor := UnaryLoggingInterceptor()

	ctx := context.Background()
	info := &grpc.UnaryServerInfo{
		FullMethod: "/test.Service/Method",
	}

	resp, err := interceptor(ctx, nil, info, mockUnaryHandler)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if resp != "success" {
		t.Errorf("Expected 'success', got %v", resp)
	}

	t.Log("✓ Logging interceptor works correctly")
}

func TestUnaryLoggingInterceptorWithError(t *testing.T) {
	interceptor := UnaryLoggingInterceptor()

	ctx := context.Background()
	info := &grpc.UnaryServerInfo{
		FullMethod: "/test.Service/Method",
	}

	_, err := interceptor(ctx, nil, info, mockErrorHandler)

	if err == nil {
		t.Fatal("Expected error from handler")
	}

	t.Log("✓ Logging interceptor logs errors correctly")
}

func TestUnaryMetricsInterceptor(t *testing.T) {
	// Reset metrics
	defaultRecorder = NewDefaultMetricsRecorder()

	interceptor := UnaryMetricsInterceptor()

	ctx := context.Background()
	info := &grpc.UnaryServerInfo{
		FullMethod: "/test.Service/Method",
	}

	_, err := interceptor(ctx, nil, info, mockUnaryHandler)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Check metrics were recorded
	metrics := defaultRecorder.GetUnaryMetrics()
	if len(metrics) != 1 {
		t.Errorf("Expected 1 metric, got %d", len(metrics))
	}

	methodMetrics, exists := metrics["/test.Service/Method"]
	if !exists {
		t.Fatal("Expected metrics for /test.Service/Method")
	}

	if methodMetrics.TotalRequests != 1 {
		t.Errorf("Expected 1 total request, got %d", methodMetrics.TotalRequests)
	}

	if methodMetrics.SuccessRequests != 1 {
		t.Errorf("Expected 1 success request, got %d", methodMetrics.SuccessRequests)
	}

	t.Log("✓ Metrics interceptor records successfully")
}

func TestUnaryMetricsInterceptorWithError(t *testing.T) {
	// Reset metrics
	defaultRecorder = NewDefaultMetricsRecorder()

	interceptor := UnaryMetricsInterceptor()

	ctx := context.Background()
	info := &grpc.UnaryServerInfo{
		FullMethod: "/test.Service/ErrorMethod",
	}

	_, err := interceptor(ctx, nil, info, mockErrorHandler)

	if err == nil {
		t.Fatal("Expected error from handler")
	}

	// Check metrics were recorded
	metrics := defaultRecorder.GetUnaryMetrics()
	methodMetrics := metrics["/test.Service/ErrorMethod"]

	if methodMetrics.TotalRequests != 1 {
		t.Errorf("Expected 1 total request, got %d", methodMetrics.TotalRequests)
	}

	if methodMetrics.FailedRequests != 1 {
		t.Errorf("Expected 1 failed request, got %d", methodMetrics.FailedRequests)
	}

	t.Log("✓ Metrics interceptor records failures correctly")
}

func TestSimpleRateLimiter(t *testing.T) {
	limiter := NewSimpleRateLimiter(5)
	defer limiter.Stop()

	userID := "user-123"
	method := "/test.Service/Method"

	// Should allow first 5 requests
	for i := 0; i < 5; i++ {
		if !limiter.Allow(userID, method) {
			t.Errorf("Expected request %d to be allowed", i+1)
		}
	}

	// 6th request should be denied
	if limiter.Allow(userID, method) {
		t.Error("Expected 6th request to be denied")
	}

	t.Log("✓ Rate limiter enforces limit correctly")
}

func TestSimpleRateLimiterSlidingWindow(t *testing.T) {
	limiter := NewSimpleRateLimiter(2)
	limiter.window = 100 * time.Millisecond // Shorter window for testing
	defer limiter.Stop()

	userID := "user-123"
	method := "/test.Service/Method"

	// First 2 requests should be allowed
	if !limiter.Allow(userID, method) {
		t.Error("Expected 1st request to be allowed")
	}
	if !limiter.Allow(userID, method) {
		t.Error("Expected 2nd request to be allowed")
	}

	// 3rd request should be denied
	if limiter.Allow(userID, method) {
		t.Error("Expected 3rd request to be denied")
	}

	// Wait for window to expire
	time.Sleep(150 * time.Millisecond)

	// Should allow requests again
	if !limiter.Allow(userID, method) {
		t.Error("Expected request after window to be allowed")
	}

	t.Log("✓ Rate limiter sliding window works correctly")
}

func TestSimpleRateLimiterPerUserMethod(t *testing.T) {
	limiter := NewSimpleRateLimiter(2)
	defer limiter.Stop()

	user1 := "user-1"
	user2 := "user-2"
	method1 := "/test.Service/Method1"
	method2 := "/test.Service/Method2"

	// User 1, Method 1: 2 requests
	if !limiter.Allow(user1, method1) {
		t.Error("Expected user1/method1 request 1 to be allowed")
	}
	if !limiter.Allow(user1, method1) {
		t.Error("Expected user1/method1 request 2 to be allowed")
	}
	if limiter.Allow(user1, method1) {
		t.Error("Expected user1/method1 request 3 to be denied")
	}

	// User 2, Method 1: should still be allowed
	if !limiter.Allow(user2, method1) {
		t.Error("Expected user2/method1 request to be allowed")
	}

	// User 1, Method 2: should still be allowed
	if !limiter.Allow(user1, method2) {
		t.Error("Expected user1/method2 request to be allowed")
	}

	t.Log("✓ Rate limiter tracks per user/method correctly")
}

func TestUnaryRateLimitInterceptor(t *testing.T) {
	limiter := NewSimpleRateLimiter(1)
	defer limiter.Stop()

	interceptor := UnaryRateLimitInterceptor(limiter)

	// Create context with user ID
	ctx := context.WithValue(context.Background(), userIDKey, "user-123")

	info := &grpc.UnaryServerInfo{
		FullMethod: "/test.Service/Method",
	}

	// First request should succeed
	_, err := interceptor(ctx, nil, info, mockUnaryHandler)
	if err != nil {
		t.Fatalf("Expected first request to succeed, got %v", err)
	}

	// Second request should be rate limited
	_, err = interceptor(ctx, nil, info, mockUnaryHandler)
	if err == nil {
		t.Fatal("Expected second request to be rate limited")
	}

	st, ok := status.FromError(err)
	if !ok {
		t.Fatal("Expected gRPC status error")
	}

	if st.Code() != codes.ResourceExhausted {
		t.Errorf("Expected ResourceExhausted code, got %v", st.Code())
	}

	t.Log("✓ Rate limit interceptor works correctly")
}

func TestMetricsRecorder(t *testing.T) {
	recorder := NewDefaultMetricsRecorder()

	// Record some metrics
	recorder.RecordUnaryRPC("/test.Service/Method1", codes.OK, 100*time.Millisecond)
	recorder.RecordUnaryRPC("/test.Service/Method1", codes.OK, 200*time.Millisecond)
	recorder.RecordUnaryRPC("/test.Service/Method1", codes.Internal, 150*time.Millisecond)

	metrics := recorder.GetUnaryMetrics()
	methodMetrics := metrics["/test.Service/Method1"]

	if methodMetrics.TotalRequests != 3 {
		t.Errorf("Expected 3 total requests, got %d", methodMetrics.TotalRequests)
	}

	if methodMetrics.SuccessRequests != 2 {
		t.Errorf("Expected 2 success requests, got %d", methodMetrics.SuccessRequests)
	}

	if methodMetrics.FailedRequests != 1 {
		t.Errorf("Expected 1 failed request, got %d", methodMetrics.FailedRequests)
	}

	if methodMetrics.MinDuration != 100*time.Millisecond {
		t.Errorf("Expected min duration 100ms, got %v", methodMetrics.MinDuration)
	}

	if methodMetrics.MaxDuration != 200*time.Millisecond {
		t.Errorf("Expected max duration 200ms, got %v", methodMetrics.MaxDuration)
	}

	expectedTotal := 450 * time.Millisecond
	if methodMetrics.TotalDuration != expectedTotal {
		t.Errorf("Expected total duration %v, got %v", expectedTotal, methodMetrics.TotalDuration)
	}

	t.Log("✓ Metrics recorder tracks correctly")
}

func TestMetricsRecorderStreamRPC(t *testing.T) {
	recorder := NewDefaultMetricsRecorder()

	recorder.RecordStreamRPC("/test.Service/StreamMethod", codes.OK, 500*time.Millisecond, "bidi")
	recorder.RecordStreamRPC("/test.Service/StreamMethod", codes.OK, 600*time.Millisecond, "bidi")

	metrics := recorder.GetStreamMetrics()
	key := "/test.Service/StreamMethod:bidi"
	streamMetrics := metrics[key]

	if streamMetrics.TotalRequests != 2 {
		t.Errorf("Expected 2 total requests, got %d", streamMetrics.TotalRequests)
	}

	if streamMetrics.SuccessRequests != 2 {
		t.Errorf("Expected 2 success requests, got %d", streamMetrics.SuccessRequests)
	}

	t.Log("✓ Stream metrics recorder works correctly")
}

func TestHelperFunctions(t *testing.T) {
	// Test getRequestID
	ctx := context.WithValue(context.Background(), requestIDKey, "test-id-123")
	id := getRequestID(ctx)
	if id != "test-id-123" {
		t.Errorf("Expected 'test-id-123', got %s", id)
	}

	// Test getUserID
	ctx = context.WithValue(context.Background(), userIDKey, "user-456")
	userID := getUserID(ctx)
	if userID != "user-456" {
		t.Errorf("Expected 'user-456', got %s", userID)
	}

	// Test anonymous user
	userID = getUserID(context.Background())
	if userID != "anonymous" {
		t.Errorf("Expected 'anonymous', got %s", userID)
	}

	t.Log("✓ Helper functions work correctly")
}

func TestExtractOrGenerateRequestID(t *testing.T) {
	// Test with existing request ID in metadata
	md := metadata.New(map[string]string{
		"x-request-id": "existing-123",
	})
	ctx := metadata.NewIncomingContext(context.Background(), md)
	id := extractOrGenerateRequestID(ctx)

	if id != "existing-123" {
		t.Errorf("Expected 'existing-123', got %s", id)
	}

	// Test without metadata - should generate new ID
	ctx = context.Background()
	id = extractOrGenerateRequestID(ctx)

	if id == "" {
		t.Error("Expected generated ID, got empty string")
	}

	t.Log("✓ Request ID extraction/generation works correctly")
}
