# Java Plugin

Install and manage Java Development Kit (JDK) for Genesis environments.

## Overview

The Java plugin provides comprehensive Java Development Kit installation and management for Genesis environments. It supports both OpenJDK and Oracle JDK distributions across macOS, Linux, and Windows platforms.

## Features

- **OpenJDK Support**: Eclipse Temurin (Adoptium) distributions
- **Oracle JDK Support**: Manual installation guidance for Oracle JDK
- **Cross-Platform Support**: Works on macOS, Linux, and Windows
- **Version Management**: Install specific Java versions (8, 11, 17, 21, etc.)
- **Environment Setup**: Automatic JAVA_HOME and PATH configuration guidance
- **Task Deduplication**: Optimized system package installation

## Installation

### Basic Configuration

```typescript
import { defineConfig } from "@ossl/genesis-core";
import { java } from "@ossl/genesis-plugins/java";

export default defineConfig({
  languages: [
    java({
      version: "17",
      distribution: "openjdk",
    }),
  ],
});
```

### YAML Configuration

```yaml
languages:
  - type: java
    version: "17"
    distribution: "openjdk"
```

## Options

| Option | Type | Required | Default | Description |
|---------|--------|----------|-------------|
| `version` | string | Yes | - | Java version to install (e.g., "8", "11", "17", "21") |
| `distribution` | string | No | "openjdk" | Distribution type: "openjdk" or "oracle" |

## Supported Versions

- **Java 8**: LTS version (legacy support)
- **Java 11**: LTS version (widely used)
- **Java 17**: LTS version (recommended)
- **Java 21**: Latest LTS version
- **Latest versions**: Automatically installs newest available

## Platform Support

### macOS & Linux (OpenJDK)

The plugin downloads official OpenJDK distributions from Adoptium:

1. Downloads appropriate tar.gz archive for architecture
2. Extracts to `/usr/local/java` (macOS) or `/opt/java` (Linux)
3. Sets up `JAVA_HOME` and `PATH` environment variables
4. Cleans up previous installations

### Windows (Oracle JDK)

Provides detailed manual installation guidance:

1. Downloads Oracle JDK from official website
2. Step-by-step installation instructions
3. Environment variable setup guidance
4. Verification steps

## Environment Variables

After installation, set up your Java environment:

```bash
# Add to your shell profile (~/.bashrc, ~/.zshrc, etc.)
export JAVA_HOME="/usr/local/java/jdk-17"
export PATH="$JAVA_HOME/bin:$PATH"
```

## Usage Examples

### Install Java 17 (Recommended)

```typescript
import { defineConfig } from "@ossl/genesis-core";
import { java } from "@ossl/genesis-plugins/java";

export default defineConfig({
  languages: [
    java({
      version: "17",
    }),
  ],
});
```

### Install Java 11 for Legacy Projects

```typescript
import { defineConfig } from "@ossl/genesis-core";
import { java } from "@ossl/genesis-plugins/java";

export default defineConfig({
  languages: [
    java({
      version: "11",
    }),
  ],
});
```

### Install Oracle JDK

```typescript
import { defineConfig } from "@ossl/genesis-core";
import { java } from "@ossl/genesis-plugins/java";

export default defineConfig({
  languages: [
    java({
      version: "21",
      distribution: "oracle",
    }),
  ],
});
```

### Multiple Java Versions

```typescript
import { defineConfig } from "@ossl/genesis-core";
import { java } from "@ossl/genesis-plugins/java";

export default defineConfig({
  languages: [
    java({
      version: "11",
    }),
    java({
      version: "17",
    }),
  ],
});
```

## Development Workflow

After installation:

```bash
# Verify Java installation
java -version
javac -version

# Check JAVA_HOME
echo $JAVA_HOME

# Compile Java code
javac MyProgram.java

# Run Java programs
java MyProgram

# Use Maven or Gradle
mvn clean install
gradle build
```

## Troubleshooting

### Java Not Found After Installation

1. Ensure `JAVA_HOME` is set correctly
2. Add `$JAVA_HOME/bin` to your PATH
3. Restart your terminal or source your shell profile
4. Check installation directory: `ls -la /usr/local/java`

### Version Mismatch

1. Check which Java is active: `java -version`
2. Verify only one Java installation exists
3. Remove previous versions before installing new ones

### Permission Issues

1. Use sudo for system-wide installation
2. Ensure write permissions to `/usr/local` or `/opt`
3. Consider user-specific installation if needed

### Windows Installation

1. Download from https://www.oracle.com/java/technologies/downloads/
2. Run the installer with administrator privileges
3. Set JAVA_HOME in System Properties
4. Add %JAVA_HOME%\bin to PATH

## Performance Notes

- OpenJDK distributions are optimized for performance
- Archive extraction is fast and efficient
- Task deduplication prevents redundant package manager updates
- Parallel installation with other plugins for optimal performance

## Security Considerations

- Downloads are from official OpenJDK/Oracle sources
- HTTPS downloads ensure secure transfer
- Archive integrity is maintained
- Regular security updates available

## Integration with Development Tools

The Java plugin integrates seamlessly with:
- **Maven**: Build automation and dependency management
- **Gradle**: Modern build tool with Groovy/Kotlin DSL
- **IntelliJ IDEA**: Full IDE support
- **Eclipse**: Open-source IDE support
- **VS Code**: Java Extension Pack
- **Spring Boot**: Framework development
- **Android Studio**: Mobile development

## Advanced Configuration

### Custom Installation Directory

For custom installation paths, modify the plugin or use manual installation:

```bash
# Install to custom directory
sudo tar -xzf openjdk-17.tar.gz -C /custom/path/
export JAVA_HOME="/custom/path/jdk-17"
```

### Multiple Java Versions

Use version managers for multiple Java versions:

```bash
# Using SDKMAN
curl -s "https://get.sdkman.io" | bash
sdk install java 17.0.2-tem
sdk use java 17.0.2-tem

# Using jenv
brew install jenv
jenv add /usr/local/java/jdk-17
jenv global 17.0
```

## Best Practices

1. **Use LTS Versions**: Prefer Java 11, 17, or 21 for production
2. **Set JAVA_HOME**: Always configure JAVA_HOME properly
3. **Version Consistency**: Ensure team uses same Java version
4. **Security Updates**: Keep Java updated for security patches
5. **Clean Installation**: Remove old versions before upgrading

## Related Plugins

- **Maven Plugin**: For build automation
- **Gradle Plugin**: For modern build systems
- **Node.js Plugin**: For full-stack development
- **Docker Plugin**: For containerized Java applications
