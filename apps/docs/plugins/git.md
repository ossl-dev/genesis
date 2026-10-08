# Git Plugin

```yaml
tools:
  - type: git
    version: "2.53.0"
    install_method: source
    install_dir: "${HOME}/.genesis/git"
```

Import `git(options = {})` from `@ossl/genesis-plugins/git`.

| Option | Default | Meaning |
| --- | --- | --- |
| `version` | `latest` | Accept any installed version, or require numeric version components |
| `install_method` | `package` | System package, Unix source build, or Windows MinGit binary |
| `install_dir` | `~/.genesis/git` | Absolute managed directory for source/binary installs |

Matching installations skip provisioning. `latest` accepts any detected version; when installation is needed, source/binary mode resolves a stable release. An explicit `install_dir` requires Git at that location. Managed installs expose their executable directory to later plugins, repositories, and scripts; persist it in your shell PATH for future sessions.

Package mode installs `git` through brew/APT/DNF/pacman/APK. Repositories determine available versions; apply fails if the installed version does not match. Use source mode for Unix release pins.

Source mode downloads [Git release tarballs](https://git-scm.com/install/source), verifies the kernel.org SHA-256 manifest, and builds relocatable command-line tools. Shared tasks install build dependencies; macOS also needs the Xcode command-line tools. GUI and localized messages are excluded; HTTPS uses libcurl. Compilation requires a temporary directory without whitespace (`TMPDIR`); the final installation path can contain spaces.

On Windows, choose `install_method: binary` for official [MinGit](https://gitforwindows.org/mingit.html) x64/ARM64 ZIP assets with published SHA-256 digests. MinGit omits Git Bash, GUI, and some interactive/Perl features. Older releases without asset digests fail explicitly. Unix binary mode is unsupported because upstream Git publishes source tarballs rather than standalone Unix binaries.

Source and binary installs verify the version and bundled HTTPS helper before and after promotion. Failed replacements restore the previous directory. Installation never changes global Git configuration. Configure identity and preferences explicitly in a setup script.

Repository cloning uses the config's [repositories section](/guide/configuration).
