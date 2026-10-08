# Bun Plugin

Status: pinned archive installation for macOS, Linux, and Windows on x64/ARM64 where the requested release provides an asset.

```yaml
tools:
  - type: bun
    version: "1.3.2"
    libc: auto
    # install_dir: "${HOME}/.local/share/genesis/bun"
```

Import `bun(options)` from `@ossl/genesis-plugins/bun`.

- `version`: required full numeric release, such as `1.3.2`.
- `install_dir`: optional absolute directory owned by this installation; defaults to `~/.genesis/bun`.
- `libc`: `auto` (default), `glibc`, or `musl`; applies to Linux archive selection. Automatic selection checks Alpine and the runtime's libc report.

The x64 installer selects baseline assets for compatibility with older Bun releases. Missing architecture/version combinations fail rather than selecting another architecture. Published asset digests or `SHASUMS256.txt` verify the streamed download. Staged and promoted `bun --version` must match before the old install is discarded. Failed promotion restores the previous directory; recovery errors retain the backup path.

Existing matching installations skip download. An explicit `install_dir` selects that managed location. Genesis rediscovers managed binaries, sets runtime `BUN_INSTALL`, and adds `bin` to `PATH`. Shell profiles and completions are not changed. The host needs `unzip` on macOS/Linux or ZIP-capable `tar` on Windows. See the upstream [Bun installation requirements](https://bun.com/docs/installation) and [platform validation limits](/guide/platform-support).
