# Plugin Overview

Genesis includes seven plugin implementations. Their available installation paths differ; detection works with preinstalled tools even where automatic installation is incomplete.

| Plugin | Automatic path | Limits |
| --- | --- | --- |
| [Node](/plugins/node) | NVM on macOS/Linux | Standalone/Windows install incomplete |
| [Python](/plugins/python) | brew or APT | Requested package and execution PATH must match |
| [Go](/plugins/go) | macOS/Linux archive | Full version and `/usr/local` permissions required |
| [Java](/plugins/java) | Experimental archive | Release/build resolution incomplete |
| [Git](/plugins/git) | brew or APT | Source/binary paths experimental |
| [Docker](/plugins/docker) | Colima or Linux installer | Desktop requires manual completion; Linux distro support partial |
| [Homebrew](/plugins/homebrew) | macOS installer | Bootstrap before shared brew tasks remains unfinished |

Windows installation generally requires manual steps. Matching runtimes skip system prerequisite work. Options are validated at load time; unsupported options fail before provisioning. Configured plugin IDs must be unique.

Bun, Deno, Rust, pnpm/Yarn, databases, mobile SDKs, IDE integration, and other proposed plugins are roadmap work rather than implemented plugin exports.

See [Lifecycle](/plugins/lifecycle) and [Creating a Plugin](/plugins/creating-plugin).
