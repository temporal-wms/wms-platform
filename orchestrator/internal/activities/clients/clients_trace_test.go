package clients

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

func TestDoRequest_PropagatesTraceContext(t *testing.T) {
	// Setup tracer with test provider
	tp := sdktrace.NewTracerProvider()
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.TraceContext{})

	tracer := tp.Tracer("test")
	ctx, span := tracer.Start(context.Background(), "test-span")
	defer span.End()

	// Mock server captures headers
	var captured http.Header
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		captured = r.Header.Clone()
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{}`))
	}))
	defer server.Close()

	// Execute request
	config := &Config{
		OrderServiceURL: server.URL,
	}
	client := NewServiceClients(config)
	err := client.doRequest(ctx, "GET", server.URL+"/test", nil, nil)

	// Verify traceparent header present and contains trace ID
	require.NoError(t, err)
	assert.NotEmpty(t, captured.Get("traceparent"), "traceparent header should be present")

	// Verify trace ID is in traceparent header
	traceID := span.SpanContext().TraceID().String()
	assert.Contains(t, captured.Get("traceparent"), traceID, "traceparent should contain trace ID")
}

func TestDoRequest_WorksWithoutSpan(t *testing.T) {
	// Test graceful handling when no span in context
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{}`))
	}))
	defer server.Close()

	config := &Config{
		OrderServiceURL: server.URL,
	}
	client := NewServiceClients(config)
	err := client.doRequest(context.Background(), "GET", server.URL+"/test", nil, nil)

	// Should work fine without span
	require.NoError(t, err, "request should succeed without span in context")
}

func TestDoRequest_PropagatesTraceWithBody(t *testing.T) {
	// Setup tracer
	tp := sdktrace.NewTracerProvider()
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.TraceContext{})

	tracer := tp.Tracer("test")
	ctx, span := tracer.Start(context.Background(), "test-span")
	defer span.End()

	// Mock server
	var captured http.Header
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		captured = r.Header.Clone()
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"success"}`))
	}))
	defer server.Close()

	// Execute POST request with body
	config := &Config{
		OrderServiceURL: server.URL,
	}
	client := NewServiceClients(config)
	body := map[string]string{"test": "value"}
	var result map[string]interface{}
	err := client.doRequest(ctx, "POST", server.URL+"/test", body, &result)

	// Verify
	require.NoError(t, err)
	assert.NotEmpty(t, captured.Get("traceparent"), "traceparent should be present in POST request")
	assert.Equal(t, "application/json", captured.Get("Content-Type"), "Content-Type should be set")
	assert.Equal(t, "application/json", captured.Get("Accept"), "Accept should be set")
}

func TestDoRequest_TraceparentFormat(t *testing.T) {
	// Setup tracer
	tp := sdktrace.NewTracerProvider()
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.TraceContext{})

	tracer := tp.Tracer("test")
	ctx, span := tracer.Start(context.Background(), "test-span")
	defer span.End()

	// Mock server
	var traceparent string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		traceparent = r.Header.Get("traceparent")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{}`))
	}))
	defer server.Close()

	// Execute
	config := &Config{
		OrderServiceURL: server.URL,
	}
	client := NewServiceClients(config)
	err := client.doRequest(ctx, "GET", server.URL+"/test", nil, nil)

	// Verify W3C Trace Context format: 00-{trace-id}-{span-id}-{flags}
	require.NoError(t, err)
	assert.Regexp(t, `^00-[0-9a-f]{32}-[0-9a-f]{16}-[0-9a-f]{2}$`, traceparent,
		"traceparent should match W3C Trace Context format")
}

func TestDoRequest_PreservesExistingHeaders(t *testing.T) {
	// Setup tracer
	tp := sdktrace.NewTracerProvider()
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.TraceContext{})

	tracer := tp.Tracer("test")
	ctx, span := tracer.Start(context.Background(), "test-span")
	defer span.End()

	// Mock server
	var captured http.Header
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		captured = r.Header.Clone()
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{}`))
	}))
	defer server.Close()

	// Execute
	config := &Config{
		OrderServiceURL: server.URL,
	}
	client := NewServiceClients(config)
	body := map[string]string{"test": "value"}
	err := client.doRequest(ctx, "POST", server.URL+"/test", body, nil)

	// Verify both trace and standard headers are present
	require.NoError(t, err)
	assert.NotEmpty(t, captured.Get("traceparent"), "traceparent should be present")
	assert.Equal(t, "application/json", captured.Get("Content-Type"), "Content-Type should be preserved")
	assert.Equal(t, "application/json", captured.Get("Accept"), "Accept should be preserved")
}
