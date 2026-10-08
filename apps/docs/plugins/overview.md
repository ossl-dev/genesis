# Plugin Overview

Genesis includes eleven plugin implementations. Installation paths and validation coverage differ; each plugin documents its limits.

| Plugin | Automatic path | Limits |
| --- | --- | --- |
| [Node](/plugins/node) | NVM on macOS/Linux | Standalone/Windows install incomplete |
| [Bun](/plugins/bun) | Checksummed, staged archive | Published x64/ARM64 assets; Linux libc selection |
| [Deno](/plugins/deno) | Checksummed, staged archive | Published x64/ARM64 assets; glibc Linux required |
| [pnpm](/plugins/pnpm) | Staged npm prefix | Compatible Node/npm required |
| [Yarn](/plugins/yarn) | Staged npm prefix, Classic or modern | Compatible Node/npm required; no Corepack integration |
| [Python](/plugins/python) | brew or APT | Requested package and execution PATH must match |
| [Go](/plugins/go) | Checksummed, staged archive | Full version; default Unix directories need permissions |
| [Java](/plugins/java) | Checksummed Temurin archive | Oracle JDK manual; published x64/ARM64 builds |
| [Git](/plugins/git) | brew or APT | Source/binary paths experimental |
| [Docker](/plugins/docker) | Colima or Linux installer | Desktop manual; Linux distro support partial |
| [Homebrew](/plugins/homebrew) | macOS installer | Same-config bootstrap precedes shared tasks |

Go, Java, Bun, and Deno have Windows archive paths; pnpm/Yarn use Windows npm shims. Other missing Windows tools generally require manual installation. An installer smoke workflow exercises pinned versions in temporary directories on macOS, Ubuntu, and Windows; local successful runs are not proof that every release/architecture works.

Matching runtimes skip installation. Options validate at load time; IDs must be unique. Rust, databases, mobile SDKs, and other proposed plugins remain [roadmap work](https://github.com/ossl-dev/genesis/blob/main/ROADMAP.md).

See [Lifecycle](/plugins/lifecycle), [Platform Support](/guide/platform-support), and [Creating a Plugin](/plugins/creating-plugin).
