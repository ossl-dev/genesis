# Installation

Build with Go 1.27 or newer; use a supported Go release for production builds.

```sh
git clone https://github.com/ossl-dev/genesis.git
cd genesis
go build -trimpath -o bin/genesis ./apps/cli
./bin/genesis --help
```

The binary runs independently of the source tree and Go compiler. Add its
location to PATH, then use it from the environment repository. Windows builds
can use `go build -o bin/genesis.exe ./apps/cli`.

```sh
genesis init --format yaml
genesis apply --dry-run
genesis apply
genesis doctor
```

`go run ./scripts/release` cross-compiles macOS/Linux/Windows amd64 and arm64.
Version-tag CI attaches these binaries to releases. Native tests and installer
coverage are described under [platform support](/guide/platform-support).
YAML is self-contained; `.go` configurations require Go at runtime. There is
no maintained public install script or package-manager distribution yet.
