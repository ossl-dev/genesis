# Deno Plugin

Status: pinned archive installation for macOS, glibc Linux, and Windows on x64/ARM64 where the requested release provides an asset. Older Windows ARM releases may be unavailable; musl Linux is explicitly rejected.

```yaml
tools:
  - type: deno
    version: "2.5.4"
    # install_dir: "${HOME}/.local/share/genesis/deno"
```

Import `deno(options)` from `@ossl/genesis-plugins/deno`.

`version` requires a full numeric release. `install_dir` is an optional absolute directory owned by this installation; the default is `~/.genesis/deno`. An explicit directory selects that managed location rather than a matching executable elsewhere on PATH.

The installer fetches the official release ZIP and its `.sha256sum`, streams and checks the archive, verifies the staged `deno --version`, then promotes it with recovery on failure. Managed installs are rediscovered on later runs. Genesis sets runtime `DENO_INSTALL` and adds its `bin` directory to `PATH`; set your shell PATH separately for future sessions. Matching versions skip installation. No shell profiles are edited.

The host needs `unzip` on macOS/Linux or ZIP-capable `tar` on Windows. Upstream release availability and OS requirements still apply; see [Deno installation](https://docs.deno.com/runtime/getting_started/installation/) and [platform validation limits](/guide/platform-support).
