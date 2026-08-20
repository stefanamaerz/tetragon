# Agent notes for Tetragon

Hard-earned context for OpenCode sessions working in this repo.

## Quick orientation

- Go 1.26 (see `go.mod`), eBPF/C, Protobuf/gRPC, Kubernetes, Helm.
- Vendored dependencies (`vendor/`). The repo replaces two local modules:
  `github.com/cilium/tetragon/api => ./api` and `github.com/cilium/tetragon/pkg/k8s => ./pkg/k8s`.
- Main binaries: `cmd/tetragon/` (agent), `cmd/tetra/` (CLI), `operator/` (K8s operator).

## Build essentials

- Minimal build: `make tetragon tetragon-bpf tetra`.
- Full default build: `make` (also compiles tests, bench, tools).
- BPF compiles in a Docker container by default; outputs to `bpf/objs/`.
  - Use `LOCAL_CLANG=1` to build with locally installed `clang`/`llc`.
  - Use `LOCAL_CLANG_FORMAT=1` for local `clang-format`.
- Runtime needs `--bpf-lib bpf/objs` (or set `TETRAGON_LIB`).
- Container engine defaults to Docker; override with `CONTAINER_ENGINE` (e.g. Podman/nerdctl).
- macOS/Apple Silicon users: develop inside a Linux VM (Lima is documented). Apple clang cannot compile BPF objects.

## nok8s build tag

- Full build has Kubernetes support. A `nok8s` tag strips it out.
- Build: `make tetragon-nok8s tetra-nok8s`.
- Many files use `//go:build !nok8s` / `//go:build nok8s`. If your editor/tooling shows missing symbols, set `GOFLAGS=-tags=nok8s`.
- CI sanity-checks: `go vet -tags nok8s ./pkg/sensors ./pkg/policyfilter ./pkg/tracingpolicy` and `go test -tags nok8s ./pkg/tracingpolicy`.

## Running tests

- `make test` runs Go tests with `sudo`, `-p 1 -parallel 1`, over `./pkg/... ./cmd/... ./operator/...`.
  It auto-builds `tester-progs` and `tetragon-bpf` first.
- Pass extra flags via `EXTRA_TESTFLAGS=...`.
- Sensor/eBPF tests live in `pkg/sensors/exec/`, `pkg/sensors/tracing/`, `pkg/sensors/test/`.
  - They require Linux, root, BTF, and compiled BPF objects.
  - They use a package-level `TestMain` calling `tus.TestSensorsRun(m, "Sensor...")`.
  - Many are gated `//go:build !windows` (some `//go:build linux`).
- Run a single sensor test:
  ```
  go test -exec sudo -p 1 -parallel 1 ./pkg/sensors/exec -run TestObserverSingle -args -bpf-lib=$(pwd)/bpf/objs
  ```
- Race detector subset: `make test-race` (controlled by `RACE_PKGS`; current CI is advisory).
- BPF unit tests: `make bpf-test [BPFGOTESTFLAGS="-v"]`.
- E2E tests: `make e2e-test` (needs kind/kubectl/helm/Docker); single package via `E2E_TESTS=...`.
- Multi-kernel VM tests: see `tests/vmtests/README.md`; requires `make test-compile` first.

## Lint, format, and generated files

- Lint: `make check` (golangci-lint v2.12.2; containerized if local version mismatches).
- Format: `make format` runs `go-format` and `clang-format`.
  - Go imports use local prefixes: `github.com/cilium/tetragon`, `github.com/cilium/tetragon/api`, `github.com/cilium/tetragon/pkg/k8s`.
- After dependency changes: `make vendor` (tidies/vendors root, `api/`, `pkg/k8s/`, and `contrib/tetragon-rthooks/`).
- Regenerate after touching specific areas:
  - `api/` protobufs: `make protogen`
  - `pkg/k8s/` CRDs/kubebuilder: `make crds`
  - Metrics reference: `make metrics-docs`
  - Daemon flags reference: `make generate-flags`
  - Tracing policy reference: `make tracing-policy-docs`
  - Helm values reference: `make -C install/kubernetes docs`
- Heavy gate: `make validate` runs linters, formatters, generators, and vendoring. Useful before push, but slow.
- Do not hand-edit generated `.pb.go`, `zz_generated.deepcopy.go`, or CRD YAMLs; regenerate instead.
- If you change CRD YAMLs in `pkg/k8s/apis/cilium.io/client/crds/v1alpha1/`, bump `CustomResourceDefinitionSchemaVersion` in `pkg/k8s/apis/cilium.io/v1alpha1/version.go`.

## CI / PR expectations

- Go tests run on `ubuntu-24.04` (amd64 + arm64) and a Windows build smoke test runs.
- Static checks enforce: every commit builds, no binary checkins, golangci-lint, gofmt, clang-format, vendoring, nok8s build.
- BPF changes trigger veristat and checkpatch workflows.
- PRs must have a `release-note/*` label and signed-off commits.
- Generated files must be up to date in CI; run the relevant `make` targets before committing.

## Common gotchas

- `make test` needs root because it loads BPF programs; use `SUDO=sudo` (default) or `SUDO="sudo -E"` to preserve env.
- `TETRAGON_LIB` env variable (or `-bpf-lib` flag) overrides the default BPF object path.
- Tests that fail may leave artifacts in `/tmp/tetragon.gotest*` and `/tmp/tetragon-bugtool*`.
- For a clean/go-fast iteration, `make tetragon-bpf tester-progs` then run focused `go test` packages.
