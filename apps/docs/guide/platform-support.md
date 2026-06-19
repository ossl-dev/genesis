# Platform Support

Genesis works across macOS, Linux, and Windows with platform-specific implementations.

## Supported Platforms

### macOS ✅

**Fully Supported**

- **Package Manager**: Homebrew
- **Shell Integration**: bash, zsh
- **Architecture**: Intel (x86_64), Apple Silicon (arm64)

**Features:**
- Automatic Homebrew detection
- Native installer support
- Shell profile integration
- Full automation

### Linux ✅

**Fully Supported**

**Debian/Ubuntu:**
- Package Manager: APT
- Automatic distro detection
- Full automation

**RedHat/Fedora:**
- Package Manager: YUM/DNF
- Automatic distro detection
- Full automation

**Features:**
- Multiple package manager support
- Shell profile integration (bash, zsh)
- Sudo handling
- Full automation

### Windows ⚠️

**Partially Supported**

- **Current**: Manual installation guides
- **Planned**: Chocolatey integration
- **Future**: Full automation

**Current Features:**
- Detailed installation guides
- Links to official installers
- Step-by-step instructions

## Platform Detection

Genesis automatically detects your platform:

```typescript
import { getPlatform } from "@ossl/genesis-core";

const platform = getPlatform();
// Returns: "macos" | "linux" | "windows"
```

## Platform-Specific Behavior

### Package Managers

Genesis uses the appropriate package manager for each platform:

| Platform | Package Manager | Update Command |
|----------|----------------|----------------|
| macOS | Homebrew | `brew update` |
| Debian/Ubuntu | APT | `apt-get update` |
| RedHat/Fedora | YUM/DNF | `yum update` / `dnf update` |
| Windows | Manual | N/A (planned: Chocolatey) |

### Shell Integration

Genesis integrates with your shell:

| Platform | Supported Shells |
|----------|-----------------|
| macOS | bash, zsh |
| Linux | bash, zsh, fish |
| Windows | PowerShell, CMD (limited) |

### Installation Methods

Different platforms use different installation methods:

**macOS:**
- Homebrew packages
- Native .pkg installers
- Downloaded binaries

**Linux:**
- System package managers (APT, YUM, DNF)
- Downloaded binaries
- Build from source (when needed)

**Windows:**
- Manual installation guides
- Links to official installers
- Future: Chocolatey packages

## Cross-Platform Configuration

Write once, run anywhere:

```typescript
import { defineConfig } from "@ossl/genesis-core";
import { node, python } from "@ossl/genesis-plugins";

// This config works on all platforms!
export default defineConfig({
  tools: [
    node({ version: "20", use_nvm: true }),
  ],
  languages: [
    python({ version: "3.11" }),
  ],
});
```

## Platform-Specific Configuration

Need platform-specific behavior? Use conditional logic:

```typescript
import { defineConfig } from "@ossl/genesis-core";
import { node, python } from "@ossl/genesis-plugins";
import { getPlatform } from "@ossl/genesis-core";

const platform = getPlatform();

export default defineConfig({
  tools: [
    node({
      version: "20",
      // Use NVM on macOS/Linux, standalone on Windows
      use_nvm: platform !== "windows",
    }),
  ],
  languages: [
    python({
      // Different versions per platform
      version: platform === "macos" ? "3.11" : "3.10",
    }),
  ],
});
```

## Limitations

### Windows

Current limitations on Windows:
- ❌ No automatic installation (manual guides provided)
- ❌ No Chocolatey integration (planned)
- ⚠️ Limited shell integration

**Workaround:** Follow the detailed installation guides provided by Genesis.

### Linux

Minor limitations:
- ⚠️ Requires sudo for system package installation
- ⚠️ Some distros may need additional repositories

### macOS

Minor limitations:
- ⚠️ Requires Homebrew (auto-installed if missing)
- ⚠️ May require Xcode Command Line Tools

## Future Support

### Planned

- ✅ Windows Chocolatey integration
- ✅ Windows automated installation
- ✅ More Linux distros (Arch, openSUSE)
- ✅ FreeBSD support

### Under Consideration

- Docker container support
- WSL (Windows Subsystem for Linux) optimization
- Android Termux support

## What's Next?

- [Getting Started](/guide/getting-started) - Install Genesis
- [Configuration](/guide/configuration) - Configure for your platform
- [Troubleshooting](/guide/troubleshooting) - Platform-specific issues

