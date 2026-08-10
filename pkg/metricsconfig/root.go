// SPDX-License-Identifier: Apache-2.0
// Copyright Authors of Tetragon

package metricsconfig

import (
	"context"
	"errors"
	"net"
	"net/http"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/cilium/tetragon/pkg/logger"
	"github.com/cilium/tetragon/pkg/logger/logfields"
)

var (
	registry     *prometheus.Registry
	registryOnce sync.Once
)

func GetRegistry() *prometheus.Registry {
	registryOnce.Do(func() {
		registry = prometheus.NewRegistry()
	})
	return registry
}

// newMetricsServer builds the metrics server and binds its listener. Binding is
// separate from serving so that callers observe bind errors synchronously, and
// so tests can learn the address that was actually bound (e.g. when asking for
// port 0).
func newMetricsServer(address string) (*http.Server, net.Listener, error) {
	reg := GetRegistry()

	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.HandlerFor(reg, promhttp.HandlerOpts{Registry: reg}))

	listener, err := net.Listen("tcp", address)
	if err != nil {
		return nil, nil, err
	}

	return &http.Server{Handler: mux}, listener, nil
}

// serveMetrics serves on the already-bound listener until ctx is canceled, at
// which point the server is gracefully shut down. It returns immediately.
//
// Two goroutines are started and both outlive this call: one serving, one
// waiting on ctx. Nothing waits for the shutdown to finish, so a caller that
// passes a context which is never canceled (e.g. context.Background()) keeps
// them for the lifetime of the process.
func serveMetrics(ctx context.Context, server *http.Server, listener net.Listener) {
	go func() {
		if err := server.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.GetLogger().Error("Metrics server exited unexpectedly", logfields.Error, err)
		}
	}()

	go func() {
		<-ctx.Done()
		if err := server.Shutdown(context.Background()); err != nil {
			logger.GetLogger().Error("Failed to shutdown metrics server", logfields.Error, err)
		}
	}()
}

// EnableMetrics starts a Prometheus metrics HTTP server on the given address.
// It binds synchronously and returns any bind error to the caller. If the bind
// succeeds, the server runs in a background goroutine and gracefully shuts down
// when the provided context is canceled.
func EnableMetrics(ctx context.Context, address string) error {
	server, listener, err := newMetricsServer(address)
	if err != nil {
		return err
	}

	logger.GetLogger().Info("Starting metrics server", "addr", listener.Addr())
	serveMetrics(ctx, server, listener)

	return nil
}
