# Java Plugin

Install and manage Java Development Kit (JDK) for Genesis environments.

## Features

- **OpenJDK Support**: Install Eclipse Temurin (Adoptium) OpenJDK distributions
- **Oracle JDK Support**: Manual installation guidance for Oracle JDK
- **Cross-Platform Support**: Works on macOS, Linux, and Windows
- **Version Management**: Install specific Java versions (8, 11, 17, 21, etc.)
- **Environment Setup**: Automatic JAVA_HOME and PATH configuration guidance
- **Task Deduplication**: Optimized system package installation

## Installation

Add to your Genesis configuration:

### TypeScript Configuration

```typescript
import { defineConfig } from "@ossl/genesis-core";
import { java } from "@ossl/genesis-plugins/java";

export default defineConfig({
  languages: [
    java({
      version: "17",
      distribution: "openjdk", // Optional: "openjdk" (default) or "oracle"
    }),
  ],
});
```

### YAML Configuration

```yaml
languages:
  - type: java
    version: "17"
    distribution: "openjdk"  # Optional: "openjdk" (default) or "oracle"
```

## Options

- `version` (string, required): Java version to install (e.g., "8", "11", "17", "21")
- `distribution` (string, optional): Distribution type ("openjdk" or "oracle", default: "openjdk")

## Platform Support

### macOS & Linux

**OpenJDK (Recommended)**:
- Downloads Eclipse Temurin (Adoptium) distributions
- Extracts to `/usr/local/java` (macOS) or `/opt/java` (Linux)
- Sets up JAVA_HOME and PATH automatically

**Oracle JDK**:
- Provides manual installation guidance
- Requires license acceptance from Oracle

### Windows

- Provides manual installation guidance
- Recommends downloading from official sources
- Instructions for environment variable setup

## Environment Variables

After installation, set up your Java environment:

```bash
# Add to your shell profile (~/.bashrc, ~/.zshrc, etc.)
export JAVA_HOME="/usr/local/java/jdk-17"  # macOS
# or
export JAVA_HOME="/opt/java/jdk-17"         # Linux
export PATH="$JAVA_HOME/bin:$PATH"
```

## Supported Versions

- Java 8 (LTS)
- Java 11 (LTS)
- Java 17 (LTS)
- Java 21 (LTS)
- Latest versions

## Examples

### Install Java 17 (OpenJDK)

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

### Install Java 11 (Oracle JDK)

```typescript
import { defineConfig } from "@ossl/genesis-core";
import { java } from "@ossl/genesis-plugins/java";

export default defineConfig({
  languages: [
    java({
      version: "11",
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

## Troubleshooting

### Java Not Found After Installation

1. Ensure JAVA_HOME is set correctly
2. Verify JAVA_HOME/bin is in your PATH
3. Restart your terminal or source your shell profile

### Version Mismatch

1. Check which Java is active: `java -version`
2. Verify JAVA_HOME points to the correct JDK
3. Use `update-alternatives` (Linux) if multiple versions exist

### Permission Issues

1. Ensure proper permissions on installation directories
2. Use sudo for system-wide installations
3. Consider user-specific installation if system access is limited

### Windows Installation

1. Download from https://adoptium.net/ (OpenJDK) or https://www.oracle.com/java/technologies/downloads/ (Oracle)
2. Run the installer
3. Set JAVA_HOME environment variable
4. Add %JAVA_HOME%\bin to PATH

## Performance Notes

- OpenJDK downloads are faster than Oracle JDK
- Archive extraction is optimized for SSD storage
- Task deduplication prevents redundant package manager updates
- Parallel installation with other plugins for optimal performance

## Security Considerations

- OpenJDK distributions are from trusted sources (Adoptium)
- Archive downloads use HTTPS
- Oracle JDK requires license acceptance
- Regular security updates available through package managers

## Integration with Other Tools

The Java plugin integrates seamlessly with:
- Maven and Gradle build tools
- Spring Boot development
- Android development (with Android SDK)
- IDEs like IntelliJ IDEA and Eclipse

---

**Part of the [Genesis](../../../README.md) project**
