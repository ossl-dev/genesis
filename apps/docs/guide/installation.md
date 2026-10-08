# Installation

The current supported path is building the monorepo from source with Bun. Standalone binary distribution and an install endpoint are roadmap work.

```bash
git clone https://github.com/ossl-dev/genesis.git
cd genesis
bun install --frozen-lockfile
bun run build
bun apps/cli/dist/index.js --help
```

To use Genesis in another project, run the built CLI by absolute path while keeping that project as the working directory:

```bash
cd /path/to/project
bun /path/to/genesis/apps/cli/dist/index.js init
bun /path/to/genesis/apps/cli/dist/index.js apply --dry-run
```

The CLI's JavaScript entrypoint has a Node shebang. YAML configs can run under Node. TypeScript configs are imported through the runtime's loader; use Bun unless you have configured Node TypeScript loading support.

No Homebrew formula, winget manifest, apt repository, or published standalone release is supplied here. The existing standalone build scripts are experimental and do not establish a distribution pipeline.

On macOS, include the Homebrew plugin to bootstrap shared brew tasks, or preinstall brew. On Debian/Ubuntu, system package tasks invoke sudo/APT. Other Linux package-manager paths and Windows Python/Git installation remain incomplete; archive runtimes and npm-prefix plugins support Windows targets. Check [Platform Support](/guide/platform-support).
