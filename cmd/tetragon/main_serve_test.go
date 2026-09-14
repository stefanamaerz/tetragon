// SPDX-License-Identifier: Apache-2.0
// Copyright Authors of Tetragon

//go:build !windows

package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/cilium/tetragon/pkg/option"
	"github.com/cilium/tetragon/pkg/server"
	"github.com/cilium/tetragon/pkg/server/eventlog"
)

func TestServeUnixSocketCleanup(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	tmpDir := t.TempDir()
	sockPath := filepath.Join(tmpDir, "tetragon.sock")
	option.Config.ServerAddress = "unix://" + sockPath

	srv := server.NewServer(ctx, nil, nil, nil, nil, nil)
	logSrv := eventlog.New()

	stop, err := Serve(ctx, option.Config.ServerAddress, srv, logSrv)
	require.NoError(t, err)

	// The socket file must exist after Serve returns.
	_, err = os.Stat(sockPath)
	require.NoError(t, err, "expected unix socket %s to exist", sockPath)

	// Calling stop should remove the socket synchronously.
	stop()
	_, err = os.Stat(sockPath)
	require.True(t, os.IsNotExist(err), "expected unix socket %s to be removed", sockPath)

	// Stop must be safe to call more than once.
	stop()
}
