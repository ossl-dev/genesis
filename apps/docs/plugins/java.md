# Java Plugin

Status: Temurin OpenJDK archive installation for macOS, Linux, and Windows on x64/ARM64 where Adoptium publishes that release. Oracle JDK requires manual download and license acceptance.

```yaml
languages:
  - type: java
    version: "17"
    distribution: openjdk
    # install_dir: "${HOME}/.local/share/genesis/jdk-17"
```

Import `java(options)` from `@ossl/genesis-plugins/java`.

| Option | Meaning |
| --- | --- |
| `version` | Numeric release prefix: `17` accepts Java 17 updates; `17.0.9` requires that patch. Legacy `1.8` means Java 8. |
| `distribution` | `openjdk` (default) or manually installed `oracle` |
| `install_dir` | Optional absolute installation directory, owned by this installation |

Default directories are `/usr/local/java/jdk-<version>` on macOS, `/opt/java/jdk-<version>` on Linux, and `%USERPROFILE%/.genesis/java/jdk-<version>` on Windows. System paths require write permissions.

Adoptium's API resolves the actual release/build and checksum. Downloads are streamed and verified before extraction. Only the freshly extracted JDK is selected; macOS bundles use `Contents/Home`. The staged executable is checked before promotion, and a failed promotion or relocated verification restores the previous directory. Recovery failure retains the backup path in the error. Crashes can leave staging and a lock requiring manual recovery.

Genesis sets runtime `JAVA_HOME` and `PATH`, including when rediscovering an existing managed install. Set them in your shell for future sessions; profiles are not edited. The host must provide `tar` (including ZIP support on Windows). See [Platform Support](/guide/platform-support) for validation limits.
