package services

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
)

func TestAssertGRPCHealthServing(t *testing.T) {
	address, stop := startHealthServer(t, "", healthpb.HealthCheckResponse_SERVING)
	defer stop()

	if err := AssertGRPCHealth(address, "", 2*time.Second, false); err != nil {
		t.Fatalf("expected serving health check, got: %v", err)
	}
}

func TestAssertGRPCHealthNotServing(t *testing.T) {
	address, stop := startHealthServer(t, "", healthpb.HealthCheckResponse_NOT_SERVING)
	defer stop()

	err := AssertGRPCHealth(address, "", 2*time.Second, false)
	if err == nil {
		t.Fatal("expected not-serving error")
	}
	if !strings.Contains(err.Error(), "NOT_SERVING") {
		t.Fatalf("expected status in error, got: %v", err)
	}
}

func TestWaitForGRPCHealth(t *testing.T) {
	address, stop := startHealthServer(t, "worker", healthpb.HealthCheckResponse_SERVING)
	defer stop()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := WaitForGRPCHealth(ctx, address, "worker", 50*time.Millisecond, false, true, false); err != nil {
		t.Fatalf("expected ready, got: %v", err)
	}
}

func TestAssertWebSocket(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upgrader := websocket.Upgrader{}
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Errorf("upgrade: %v", err)
			return
		}
		conn.Close()
	}))
	defer srv.Close()

	if err := AssertWebSocket(toWebSocketURL(srv.URL), 2*time.Second); err != nil {
		t.Fatalf("expected WebSocket upgrade, got: %v", err)
	}
}

func TestAssertWebSocketRejectsHTTP(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	err := AssertWebSocket(toWebSocketURL(srv.URL), 2*time.Second)
	if err == nil {
		t.Fatal("expected WebSocket error")
	}
}

func TestWaitForWebSocket(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upgrader := websocket.Upgrader{}
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Errorf("upgrade: %v", err)
			return
		}
		conn.Close()
	}))
	defer srv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := WaitForWebSocket(ctx, toWebSocketURL(srv.URL), 50*time.Millisecond, false, true); err != nil {
		t.Fatalf("expected ready, got: %v", err)
	}
}

func startHealthServer(t *testing.T, serviceName string, status healthpb.HealthCheckResponse_ServingStatus) (string, func()) {
	t.Helper()

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}

	server := grpc.NewServer()
	healthServer := health.NewServer()
	healthServer.SetServingStatus(serviceName, status)
	healthpb.RegisterHealthServer(server, healthServer)

	go func() {
		if err := server.Serve(ln); err != nil {
			t.Logf("grpc serve: %v", err)
		}
	}()

	return ln.Addr().String(), func() {
		server.Stop()
		ln.Close()
	}
}

func toWebSocketURL(urlStr string) string {
	return "ws" + strings.TrimPrefix(urlStr, "http")
}
