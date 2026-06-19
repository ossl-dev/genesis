# Getting Started

**Stop wasting hours on setup. Let's get you productive in 5 minutes.**

---

## **🚀 The Easy Way (Recommended)**

### One Command Installation

```bash
# Install Genesis globally (works on fresh systems!)
curl -fsSL https://genesis-docs.vercel.app/install | sh

# That's it. No dependencies, no Node.js required.
genesis --help
```

### Your First Environment

```bash
# Create a new project
mkdir my-project
cd my-project

# Initialize Genesis config
genesis init

# Apply your environment (this is the magic)
genesis apply

# Done! Your development environment is ready.
```

---

## **📦 What Just Happened?**

When you ran `genesis apply`, Genesis:

1. **Detected** your system (macOS/Linux/Windows)
2. **Downloaded** and installed Node.js (via NVM)
3. **Set up** Python if needed
4. **Configured** Git with your details
5. **Cloned** any repositories in your config
6. **Set** environment variables
7. **Validated** everything works

**All in parallel, with zero redundant operations.**

---

## **🎯 Your First Configuration**

Genesis created `genesis.config.yaml`. Let's make it awesome:

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

---

## **⚡ The Magic Commands**

### `genesis init` - Create Your Config

```bash
genesis init                    # Interactive setup
genesis init --typescript       # TypeScript config
genesis init --defaults        # Quick start with defaults
```

### `genesis apply` - Make It Happen

```bash
genesis apply                    # Apply current config
genesis apply --verbose         # See all the details
genesis apply --dry-run          # Preview changes
genesis apply production-env-123 # Apply cloud environment
```

### `genesis diff` - Crystal Ball

```bash
genesis diff                     # See what would change
```

Output:

```
🔍 Genesis Diff

Changes that would be applied:

  node:
    Current:  Not installed
    Desired:  v20.10.0
    Action:   Install Node.js v20 via NVM

  python:
    Current:  Python 3.9.2
    Desired:  Python 3.11.5
    Action:   Install Python 3.11 via pyenv
```

### `genesis validate` - Health Check

```bash
genesis validate                 # Check everything's working
```

### `genesis doctor` - Fix Problems

```bash
genesis doctor                   # Run diagnostics
```

---

## **☁️ Cloud Features (Game Changer)**

### Connect to Genesis Cloud

```bash
genesis login                    # OAuth in browser
genesis login --token xyz123     # Or use token
```

### List Your Environments

```bash
genesis list                     # All environments
genesis list --cloud             # Cloud only
genesis list --local             # Local only
```

Output:

```
🌐 Cloud Environments:
  - production-prod-123 (Production setup)
  - staging-stage-456 (Staging setup)
  - dev-dev-789 (Development setup)

🏠 Local Environments:
  - current (./genesis.config.yaml)
  - experiment (./experiment.config.yaml)
```

### Apply Cloud Environments

```bash
genesis apply production-prod-123
```

Genesis downloads and applies the exact same setup your team uses in production.

---

## **🎨 Configuration Formats**

### YAML (Simple & Clean)

```yaml
tools:
  - type: node
    version: "20"
    use_nvm: true

env:
  NODE_ENV: development
  API_URL: http://localhost:3000
```

### TypeScript (Type Safe & Powerful)

```typescript
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
    }),
    git({
      user_name: "Your Name",
      user_email: "you@company.com",
    }),
  ],
  env: {
    NODE_ENV: "development",
    API_URL: "http://localhost:3000",
  },
});
```

---

## **🚀 Real-World Workflows**

### New Developer Onboarding (15 Minutes)

```bash
# Day 1 at new job:
git clone company-project
cd company-project

# Genesis config is already in the repo
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

---

## **🎯 Advanced Usage**

### Environment Caching

```bash
# Genesis caches environments for instant switching
# First setup: 5 minutes
# Switch to cached environment: 10 seconds

genesis apply node-project     # Sets up Node.js project
cd ../python-project
genesis apply python-project   # Instant switch to Python setup
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

## **🔧 Platform-Specific Magic**

Genesis automatically detects your platform:

### macOS

- Uses **Homebrew** for system packages
- Downloads installers from official sources
- Integrates with shell profiles (.zshrc, .bashrc)

### Linux

- Uses **APT** (Debian/Ubuntu)
- Uses **YUM/DNF** (RedHat/Fedora)
- Detects distribution automatically

### Windows

- Downloads official installers
- Configures PATH and environment
- Future: Automated installation support

---

## **🛠️ Development Setup**

If you want to contribute to Genesis:

```bash
# Clone the repository
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

---

## **🎁 Pro Tips**

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

## **🤝 Need Help?**

- **Documentation**: [Full Docs](https://genesis-docs.vercel.app)
- **Issues**: [GitHub Issues](https://github.com/your-org/genesis/issues)
- **Discussions**: [GitHub Discussions](https://github.com/your-org/genesis/discussions)
- **Community**: [Discord](https://discord.gg/genesis)

---

## **🎯 What's Next?**

Now that you have Genesis working:

1. **[Explore Cloud Features](/guide/cloud)** - Team environments and collaboration
2. **[Learn Configuration](/guide/configuration)** - Advanced configuration options
3. **[Discover Plugins](/plugins/overview)** - See what plugins are available
4. **[Build Plugins](/guide/plugin-development)** - Create your own plugins
5. **[API Reference](/api/core)** - Complete technical documentation

---

**Welcome to the future of environment setup. Your productivity is about to skyrocket.** 🚀
