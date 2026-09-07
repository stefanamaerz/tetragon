// SPDX-License-Identifier: Apache-2.0
// Copyright Authors of Tetragon

package health

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
)

// longInterval keeps the health poll ticker from firing during tests.
const longInterval = 3600

func TestServeHealthIgnoresServerStopped(t *testing.T) {
	// Stopping the server before Serve() guarantees grpc.ErrServerStopped.
	// serveHealth must treat that as a clean shutdown, not a fatal error.
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer lis.Close()

	srv := grpc.NewServer()
	srv.Stop()

	done := make(chan struct{})
	go func() {
		serveHealth(srv, lis)
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("serveHealth did not return; likely called logger.Fatal on ErrServerStopped")
	}
}

func TestStartHealthServerListenError(t *testing.T) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	// Keep the listener open while StartHealthServer tries to bind to the same
	// address, otherwise the OS may reuse the ephemeral port.
	defer l.Close()

	err = StartHealthServer(context.Background(), l.Addr().String(), longInterval)
	require.ErrorContains(t, err, "failed to listen for gRPC healthserver")
}
