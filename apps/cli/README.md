# Genesis CLI

> **The command-line tool that makes development environments magical**

Remember spending hours following setup guides that don't work? Yeah, we fixed that. The Genesis CLI is your gateway to instant, reproducible development environments.

---

## 🚀 Why This CLI Is Different

### Most CLIs Are Boring

```bash
# Other tools: cryptic commands, confusing options
some-tool configure --with-feature --enable-mode --path=/usr/local/bin
some-tool install --force --verbose
some-tool validate --check-all --report-format=json
```

### Genesis CLI Is Delightful

```bash
# Genesis: simple, intuitive, does what you mean
genesis init          # Create your environment config
genesis apply         # Make it happen
genesis diff          # See what would change
genesis validate      # Check everything's good
```

---

## 🎮 Installation (Stupidly Simple)

### The Easy Way (Recommended)

```bash
# One command, works everywhere
curl -fsSL https://genesis-docs.vercel.app/install | sh

# Now you have `genesis` globally!
genesis --help
```

### The Manual Way

```bash
# Download the standalone binary
curl -fsSL https://releases.genesis-docs.vercel.app/latest/genesis-linux-x64 -o genesis
chmod +x genesis
sudo mv genesis /usr/local/bin/

# That's it. No dependencies, no Node.js required.
```

### The Developer Way

```bash
# If you're contributing to Genesis
git clone https://github.com/your-org/genesis.git
cd genesis
bun install
bun run build
cd apps/cli
npm link
```

---

## 🎯 The Commands You'll Actually Use

### `genesis init` - Start Your Journey

```bash
# Create a new environment config
genesis init

# Answer a few questions (or skip with --defaults)
# Genesis creates genesis.config.yaml with your setup
```

**What happens:**

- Checks if you already have a config
- Asks about your stack (Node.js? Python? Docker?)
- Creates a perfect config file
- Gives you next steps

### `genesis apply` - The Magic Button

```bash
# Apply your environment configuration
genesis apply

# Genesis does everything:
# ✅ Installs Node.js (via NVM)
# ✅ Sets up Python (via pyenv)
# ✅ Configures Git with your details
# ✅ Installs Docker and starts daemon
# ✅ Clones your repositories
# ✅ Sets environment variables
# ✅ Runs your setup scripts
# ✅ Validates everything works
```

**Options that matter:**

```bash
genesis apply --config ./custom.yaml    # Use specific config
genesis apply --verbose                  # See all the details
genesis apply --dry-run                  # Preview without changes
genesis apply production-env-123         # Apply cloud environment
```

### `genesis diff` - Crystal Ball Mode

```bash
# See what would change before you apply
genesis diff

# Output:
# 🔍 Genesis Diff
#
# Changes that would be applied:
#
#   node:
#     Current:  Not installed
#     Desired:  v20.10.0
#     Action:   Install Node.js v20 via NVM
#
#   python:
#     Current:  Python 3.9.2
#     Desired:  Python 3.11.5
#     Action:   Install Python 3.11 via pyenv
```

### `genesis validate` - Health Check

```bash
# Make sure your environment matches the config
genesis validate

# Output:
# ✅ Genesis Validate
#
# ✓ node: Node.js v20.10.0 is correctly installed
# ✓ python: Python 3.11.5 is correctly installed
# ✓ git: Git is configured with your details
#
# Summary: 3/3 plugins valid
```

### `genesis doctor` - Fix Problems

```bash
# Something not working? Run the doctor
genesis doctor

# Genesis checks:
# ✓ Config file exists and is valid
# ✓ All plugins are available
# ✓ System requirements met
# ✓ No dependency conflicts
# ✓ Network connectivity
#
# And gives you specific fixes if something's wrong
```

### `genesis list-plugins` - What's Available

```bash
# See all the tools Genesis can install
genesis list-plugins

# Output:
# 📦 Available Plugins
#
# Tools:
#   node - Node.js runtime
#     version (string): Version to install
#     use_nvm (boolean): Use NVM (default: true)
#
#   python - Python interpreter
#     version (string): Version to install
#     use_pyenv (boolean): Use pyenv (default: true)
#
#   git - Git version control
#     user_name (string): Your name
#     user_email (string): Your email
#
#   docker - Docker container platform
#     auto_start (boolean): Start daemon (default: true)
```

---

## 🌟 Cloud Features (Game Changer)

### `genesis login` - Connect to the Cloud

```bash
# Login to Genesis Cloud
genesis login

# Opens browser for OAuth, or use token:
genesis login --token your-api-token
```

### `genesis list` - See Your Environments

```bash
# List all your environments
genesis list

# Local environments:
# 🏠 Local Environments:
#   - current (./genesis.config.yaml)
#   - staging (./staging.config.yaml)
#
# Cloud environments:
# 🌐 Cloud Environments:
#   - production-prod-123 (Production setup)
#   - staging-stage-456 (Staging setup)
#   - dev-dev-789 (Development setup)
```

### `genesis apply <env-id>` - Apply Cloud Environments

```bash
# Apply a specific cloud environment
genesis apply production-prod-123

# Genesis downloads and applies the exact same setup
# your team uses in production
```

---

## 🎨 Configuration Files (Your Environment as Code)

### YAML - Simple and Clean

```yaml
# genesis.config.yaml
tools:
  - type: node
    version: "20"
    use_nvm: true

  - type: python
    version: "3.11"

  - type: git
    user_name: "Your Name"
    user_email: "you@company.com"

repositories:
  - url: https://github.com/your-org/backend
    path: ./backend
    branch: main

  - url: https://github.com/your-org/frontend
    path: ./frontend
    branch: develop

env:
  NODE_ENV: development
  API_URL: http://localhost:3000
  DATABASE_URL: postgresql://localhost:5432/myapp

scripts:
  - name: setup-database
    command: docker-compose up -d postgres
    when: before

  - name: install-deps
    command: npm install
    when: after
```

### TypeScript - Type Safe and Powerful

```typescript
// genesis.config.ts
import { defineConfig } from "@ossl/genesis-core";
import { node, python, git } from "@ossl/genesis-plugins";

export default defineConfig({
  tools: [
    node({
      version: "20",
      use_nvm: true,
    }),
    python({
      version: "3.11",
      use_pyenv: true,
    }),
    git({
      user_name: "Your Name",
      user_email: "you@company.com",
    }),
  ],
  repositories: [
    {
      url: "https://github.com/your-org/backend",
      path: "./backend",
      branch: "main",
    },
  ],
  env: {
    NODE_ENV: "development",
    API_URL: "http://localhost:3000",
  },
});
```

---

## 🚀 Real-World Workflows

### New Developer Onboarding (15 Minutes)

```bash
# Day 1 at new job:
git clone company-project
cd company-project

# Genesis config is already in the repo
# Just apply it!
genesis apply

# 5 minutes later:
# ✅ All tools installed
# ✅ All repos cloned
# ✅ Environment configured
# ✅ Ready to contribute

# Validate everything's working
genesis validate
```

### Switching Between Projects

```bash
# Working on Project A (Node.js stack)
cd project-a
genesis apply

# Switch to Project B (Python stack)
cd ../project-b
genesis apply

# Genesis handles the switch perfectly
# No conflicts, no mess
```

### Team Standardization

```bash
# DevOps creates the perfect config
# Commits it to the repo

# Every developer runs:
git pull
genesis apply

# Everyone has identical environments
# No more "works on my machine" issues
```

### Updating Tools

```bash
# Team decides to upgrade Node.js 18 -> 20
# Edit genesis.config.yaml:
# version: "18" -> version: "20"

# Every developer runs:
genesis diff    # See what's changing
genesis apply   # Apply the upgrade

# Whole team upgraded in sync
```

---

## 🎯 Advanced Usage

### Environment Caching

```bash
# Genesis caches environments for instant switching
# First setup: 5 minutes
# Switch to cached environment: 10 seconds

genesis apply node-project     # Sets up Node.js project
cd ../python-project
genesis apply python-project   # Instant switch to Python setup
```

### Parallel Execution

```bash
# Genesis installs everything in parallel
# Node.js, Python, Git, Docker all install at once
# 3x faster than sequential installation
```

### Custom Scripts

```bash
# Add custom setup steps to your config
scripts:
  - name: setup-database
    command: docker-compose up -d postgres
    when: before

  - name: install-vscode-extensions
    command: code --install-extension ms-python.python
    when: after
```

---

## 🏗️ How It Works Under the Hood

### The Three-Phase Magic

1. **Detect** - What's currently installed?
2. **Apply** - What needs to change?
3. **Validate** - Did it work correctly?

### Plugin Architecture

- Each tool is a plugin (node, python, git, docker)
- Plugins handle platform differences automatically
- Extensible - build your own plugins

### Cross-Platform Magic

- Same config works on macOS, Linux, Windows
- Handles different package managers (brew, apt, yum, choco)
- Manages shell differences (bash, zsh, fish, powershell)

---

## 🎁 Pro Tips

### Speed Things Up

```bash
# Use Genesis caching
genesis apply    # First time: 5 minutes
genesis apply    # Second time: 10 seconds (cached)

# Parallel execution
# Genesis installs multiple tools simultaneously
```

### Stay Safe

```bash
# Always check before applying
genesis diff     # See what will change
genesis apply    # Apply changes
genesis validate # Verify everything works
```

### Debug Like a Pro

```bash
# Verbose mode for troubleshooting
genesis apply --verbose

# Doctor mode for health checks
genesis doctor

# List available tools
genesis list-plugins --verbose
```

---

## 🚀 Building From Source

```bash
# Clone the repo
git clone https://github.com/your-org/genesis.git
cd genesis

# Install dependencies
bun install

# Build the CLI
bun run build

# Link for local development
cd apps/cli
npm link

# Test your changes
genesis --help
```

### Project Structure

```
src/
├── commands/         # CLI command implementations
│   ├── apply.ts      # Apply configuration
│   ├── init.ts       # Initialize config
│   ├── diff.ts       # Show differences
│   ├── validate.ts   # Validate environment
│   ├── doctor.ts     # Run diagnostics
│   ├── list.ts       # List environments
│   ├── login.ts      # Cloud login
│   └── list-plugins.ts # Show available tools
├── lib/              # CLI utilities
│   └── runner.ts     # Core execution logic
└── index.ts          # CLI entry point
```

---

## 🤝 Contributing

We're building this together! Whether you're:

- 🐛 Fixing bugs
- ✨ Adding new commands
- 📖 Improving documentation
- 🔌 Building plugins

Your contribution matters. See the [Contributing Guide](../../CONTRIBUTING.md) to get started.

---

## 💬 Need Help?

- **Issues**: [GitHub Issues](https://github.com/your-org/genesis/issues)
- **Discussions**: [GitHub Discussions](https://github.com/your-org/genesis/discussions)
- **Documentation**: [Full Docs](https://genesis-docs.vercel.app)

---

## 🎯 One More Thing

This isn't just another CLI tool. It's the beginning of the end for environment setup pain. We've all wasted countless hours on setup guides that don't work, on "works on my machine" bugs, on outdated documentation.

Genesis fixes that. For good.

Try it. You'll wonder how you ever lived without it.

---

**Made with ❤️ by developers who were tired of wasting time on setup**

---

_P.S. Yes, it really is this good. Try it and see for yourself._ 🚀
