package services

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestWaitForURL_ReadyImmediately(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := WaitForURL(ctx, srv.URL, []int{200}, "", 50*time.Millisecond, false, true); err != nil {
		t.Fatalf("expected ready, got: %v", err)
	}
}

func TestWaitForURL_BodyLeakRegression(t *testing.T) {
	// Returns 503 a few times then 200, with a non-empty body.
	// Without the fix, every retry leaks a response body.
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		n := calls.Add(1)
		if n < 4 {
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write([]byte("not yet"))
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ready"))
	}))
	defer srv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := WaitForURL(ctx, srv.URL, []int{200}, "ready", 20*time.Millisecond, false, true); err != nil {
		t.Fatalf("expected ready after retries, got: %v", err)
	}
	if got := calls.Load(); got < 4 {
		t.Fatalf("expected at least 4 attempts, got %d", got)
	}
}

func TestWaitForURL_Timeout(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	err := WaitForURL(ctx, srv.URL, []int{200}, "", 50*time.Millisecond, false, true)
	if err == nil {
		t.Fatal("expected timeout error")
	}
	if !strings.Contains(err.Error(), "timeout") {
		t.Fatalf("expected timeout error, got: %v", err)
	}
}

func TestWaitForURL_BodyMatch(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("status: healthy"))
	}))
	defer srv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := WaitForURL(ctx, srv.URL, []int{200}, "healthy", 50*time.Millisecond, false, true); err != nil {
		t.Fatalf("expected match, got: %v", err)
	}

	ctx2, cancel2 := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel2()
	err := WaitForURL(ctx2, srv.URL, []int{200}, "broken", 50*time.Millisecond, false, true)
	if err == nil {
		t.Fatal("expected timeout when body doesn't match")
	}
}

func TestWaitForTCP_Ready(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer ln.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := WaitForTCP(ctx, ln.Addr().String(), 50*time.Millisecond, false, true); err != nil {
		t.Fatalf("expected ready, got: %v", err)
	}
}

func TestWaitForTCP_Timeout(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()
	// Reserved unassignable port for "discard" — refuses fast
	err := WaitForTCP(ctx, "127.0.0.1:1", 50*time.Millisecond, false, true)
	if err == nil {
		t.Fatal("expected timeout")
	}
}

func TestWaitForCommand(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := WaitForCommand(ctx, "true", 50*time.Millisecond, false, true); err != nil {
		t.Fatalf("expected success, got: %v", err)
	}

	ctx2, cancel2 := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel2()
	err := WaitForCommand(ctx2, "false", 50*time.Millisecond, false, true)
	if err == nil {
		t.Fatal("expected timeout when command always fails")
	}
}
