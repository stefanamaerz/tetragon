// SPDX-License-Identifier: Apache-2.0
// Copyright Authors of Tetragon

package health

import (
	"context"
	"errors"
	"fmt"
	"net"
	"time"

	"google.golang.org/grpc"
	gh "google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"

	"github.com/cilium/tetragon/api/v1/tetragon"
	"github.com/cilium/tetragon/pkg/logger"
	"github.com/cilium/tetragon/pkg/logger/logfields"
)

var (
	log = logger.GetLogger()
)

// StartHealthServer binds the health check listener and serves it in the
// background. It returns an error only if binding fails. Other Serve errors
// are fatal except for grpc.ErrServerStopped, which is returned when the
// server is stopped as part of a clean shutdown.
func StartHealthServer(ctx context.Context, address string, interval int) error {
	// Create a new health server and mark it as serving.
	healthServer := gh.NewServer()
	healthServer.SetServingStatus("liveness", grpc_health_v1.HealthCheckResponse_SERVING)

	// Create a new gRPC server for health checks and register the healthServer.
	grpcHealthServer := grpc.NewServer()
	grpc_health_v1.RegisterHealthServer(grpcHealthServer, healthServer)

	listener, err := net.Listen("tcp", address)
	if err != nil {
		return fmt.Errorf("failed to listen for gRPC healthserver: %w", err)
	}
	log.Info("Starting gRPC health server", "address", address, "interval", interval)

	// Serve until ctx is cancelled, at which point Stop() below makes Serve
	// return grpc.ErrServerStopped.
	go serveHealth(grpcHealthServer, listener)

	// Check the agent health periodically. To check if our agent is health we call
	// health.GetHealth() and we report the status to the healthServer.
	go func() {
		ticker := time.NewTicker(time.Duration(interval) * time.Second)
		for {
			select {
			case <-ticker.C:
				servingStatus := grpc_health_v1.HealthCheckResponse_NOT_SERVING
				if response, err := GetHealth(); err == nil {
					if st := response.GetHealthStatus(); len(st) > 0 && st[0].Status == tetragon.HealthStatusResult_HEALTH_STATUS_RUNNING {
						servingStatus = grpc_health_v1.HealthCheckResponse_SERVING
					}
				}
				healthServer.SetServingStatus("liveness", servingStatus)
			case <-ctx.Done():
				ticker.Stop()
				healthServer.Shutdown() // set all services to NOT_SERVING
				grpcHealthServer.Stop()
				return
			}
		}
	}()

	return nil
}

func serveHealth(srv *grpc.Server, lis net.Listener) {
	if err := srv.Serve(lis); err != nil && !errors.Is(err, grpc.ErrServerStopped) {
		logger.Fatal(log, "gRPC health server Serve returned error", logfields.Error, err)
	}
}
