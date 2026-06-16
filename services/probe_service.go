package services

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
)

// AssertGRPCHealth checks the standard gRPC health service once.
func AssertGRPCHealth(address string, serviceName string, timeout time.Duration, useTLS bool) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	opts := []grpc.DialOption{grpc.WithBlock()}
	if useTLS {
		opts = append(opts, grpc.WithTransportCredentials(credentials.NewClientTLSFromCert(nil, "")))
	} else {
		opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	}

	conn, err := grpc.DialContext(ctx, address, opts...)
	if err != nil {
		return fmt.Errorf("connecting to gRPC %s: %w", address, err)
	}
	defer conn.Close()

	resp, err := healthpb.NewHealthClient(conn).Check(ctx, &healthpb.HealthCheckRequest{Service: serviceName})
	if err != nil {
		return fmt.Errorf("checking gRPC health %s: %w", address, err)
	}
	if resp.GetStatus() != healthpb.HealthCheckResponse_SERVING {
		return fmt.Errorf("gRPC %s health status %s, expected SERVING", address, resp.GetStatus())
	}
	return nil
}

// AssertWebSocket checks that a WebSocket endpoint accepts an upgrade once.
func AssertWebSocket(urlStr string, timeout time.Duration) error {
	dialer := websocket.Dialer{HandshakeTimeout: timeout}
	conn, resp, err := dialer.Dial(urlStr, nil)
	if err != nil {
		if resp != nil {
			return fmt.Errorf("connecting to WebSocket %s: status %d: %w", urlStr, resp.StatusCode, err)
		}
		return fmt.Errorf("connecting to WebSocket %s: %w", urlStr, err)
	}
	defer conn.Close()

	if resp == nil || resp.StatusCode != http.StatusSwitchingProtocols {
		status := 0
		if resp != nil {
			status = resp.StatusCode
		}
		return fmt.Errorf("WebSocket %s returned status %d, expected 101", urlStr, status)
	}
	return nil
}
