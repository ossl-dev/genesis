# Installation

**Get Genesis running in 30 seconds. No dependencies required.**

---

## **🚀 The Easy Way (Recommended)**

### One Command Installation

```bash
# Install Genesis globally (works on fresh systems!)
curl -fsSL https://genesis-docs.vercel.app/install | sh

# That's it. No dependencies, no Node.js required.
genesis --help
```

**What this does:**

- Downloads the standalone Genesis binary for your platform
- Installs it to `/usr/local/bin` (or adds to your PATH)
- Makes `genesis` available globally
- Works on macOS, Linux, and Windows

### Manual Download

```bash
# Download the standalone binary for your platform
curl -fsSL https://releases.genesis-docs.vercel.app/latest/genesis-darwin-x64 -o genesis
chmod +x genesis
sudo mv genesis /usr/local/bin/

# Or for Linux
curl -fsSL https://releases.genesis-docs.vercel.app/latest/genesis-linux-x64 -o genesis
chmod +x genesis
sudo mv genesis /usr/local/bin/

# Or for Windows
# Download from https://releases.genesis-docs.vercel.app/latest/genesis-windows-x64.exe
# Add to your PATH
```

---

## **🎯 Verify Installation**

```bash
# Check Genesis is working
genesis --version

# See all available commands
genesis --help

# Test with a simple config
mkdir test-genesis
cd test-genesis
genesis init
genesis apply --dry-run
```

---

## **🔧 Platform-Specific Notes**

### macOS

```bash
# Genesis works on both Intel and Apple Silicon
# The installer automatically detects your architecture

# If you use Homebrew (optional)
brew tap genesis/tap
brew install genesis
```

### Linux

```bash
# Works on Ubuntu, Debian, Fedora, CentOS, and more
# Automatically detects your distribution

# For Ubuntu/Debian
wget -qO- https://releases.genesis-docs.vercel.app/latest/genesis-linux-x64.deb | sudo dpkg -i -

# For Fedora/CentOS
sudo rpm -i https://releases.genesis-docs.vercel.app/latest/genesis-linux-x64.rpm
```

### Windows

```bash
# Download the .exe installer
# https://releases.genesis-docs.vercel.app/latest/genesis-windows-x64.exe

# Or use Chocolatey (optional)
choco install genesis

# Or use Scoop (optional)
scoop install genesis
```

---

## **🛠️ Development Installation**

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

## **📦 Package Managers**

### npm (Global)

```bash
npm install -g @ossl/genesis-cli
```

### Bun (Global)

```bash
bun add -g @ossl/genesis-cli
```

### Yarn (Global)

```bash
yarn global add @ossl/genesis-cli
```

### pnpm (Global)

```bash
pnpm add -g @ossl/genesis-cli
```

---

## **🐳 Docker Installation**

### Using the Official Docker Image

```bash
# Pull the latest Genesis image
docker pull genesis/cli:latest

# Run Genesis commands
docker run --rm -v $(pwd):/workspace genesis/cli:latest init
docker run --rm -v $(pwd):/workspace genesis/cli:latest apply
```

### Docker Compose

```yaml
# docker-compose.yml
version: "3.8"
services:
  genesis:
    image: genesis/cli:latest
    volumes:
      - .:/workspace
    working_dir: /workspace
    command: apply
```

---

## **🔄 Update Genesis**

### Update Standalone Binary

```bash
# Re-run the installer
curl -fsSL https://genesis-docs.vercel.app/install | sh

# Or manually download the latest version
curl -fsSL https://releases.genesis-docs.vercel.app/latest/genesis-darwin-x64 -o genesis
chmod +x genesis
sudo mv genesis /usr/local/bin/
```

### Update Package Manager Installation

```bash
# npm
npm update -g @ossl/genesis-cli

# Bun
bun update -g @ossl/genesis-cli

# Yarn
yarn global upgrade @ossl/genesis-cli

# pnpm
pnpm update -g @ossl/genesis-cli
```

---

## **🗑️ Uninstall Genesis**

### Remove Standalone Binary

```bash
# Remove the binary
sudo rm /usr/local/bin/genesis

# Or if you installed it elsewhere
rm ~/.local/bin/genesis
```

### Remove Package Manager Installation

```bash
# npm
npm uninstall -g @ossl/genesis-cli

# Bun
bun remove -g @ossl/genesis-cli

# Yarn
yarn global remove @ossl/genesis-cli

# pnpm
pnpm remove -g @ossl/genesis-cli
```

### Clean Up Configuration

```bash
# Remove Genesis configuration directory
rm -rf ~/.genesis

# Remove Genesis from shell profiles
# Edit ~/.zshrc, ~/.bashrc, etc. to remove Genesis PATH entries
```

---

## **🔍 Troubleshooting**

### Command Not Found

```bash
# If you get "command not found: genesis"
# Check if Genesis is in your PATH
which genesis

# If not found, add it manually
echo 'export PATH="$PATH:/usr/local/bin"' >> ~/.zshrc
source ~/.zshrc
```

### Permission Denied

```bash
# If you get permission errors
chmod +x /usr/local/bin/genesis

# Or reinstall with proper permissions
curl -fsSL https://genesis-docs.vercel.app/install | sudo sh
```

### Network Issues

```bash
# If you can't download Genesis
# Check your internet connection
curl -I https://genesis-docs.vercel.app

# Or use a different mirror
curl -fsSL https://mirror.genesis-docs.vercel.app/install | sh
```

### Antivirus Issues

```bash
# Some antivirus software may flag Genesis
# Add Genesis to your antivirus exceptions
# Or verify the download checksum
sha256sum genesis
# Compare with the checksum on the releases page
```

---

## **🎯 Next Steps**

Now that you have Genesis installed:

1. **[Quick Start](/guide/getting-started)** - Create your first environment
2. **[Configuration](/guide/configuration)** - Learn about configuration options
3. **[Cloud Features](/guide/cloud)** - Team environments and collaboration
4. **[Plugins](/plugins/overview)** - Discover available plugins
5. **[API Reference](/api/core)** - Complete technical documentation

---

## **🤝 Need Help?**

- **Documentation**: [Full Docs](https://genesis-docs.vercel.app)
- **Issues**: [GitHub Issues](https://github.com/your-org/genesis/issues)
- **Discussions**: [GitHub Discussions](https://github.com/your-org/genesis/discussions)
- **Community**: [Discord](https://discord.gg/genesis)

---

**Ready to revolutionize your development environment?** 🚀

Let's get you started with your first Genesis environment.
