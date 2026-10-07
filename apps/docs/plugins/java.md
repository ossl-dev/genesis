# Java Plugin

Status: detection implemented; archive installation is experimental. Windows and Oracle installation are manual.

```yaml
languages:
  - type: java
    version: "17"
    distribution: openjdk
```

Import `java(options)` from `@ossl/genesis-plugins/java`. `version` is required and numeric; `distribution` accepts `openjdk` (default) or `oracle`.

Detection checks `java -version`, including stderr and older `1.8` version syntax. It compares major versions. Matching installations skip system prerequisites.

The OpenJDK installer constructs an Adoptium release archive URL and extracts into `/usr/local/java` on macOS or `/opt/java` on Linux. Release/build resolution is incomplete: `17` can detect a preinstalled JDK but is not a valid archive release identifier. Install a JDK manually until release resolution is completed.

The installer prints JAVA_HOME/PATH instructions instead of editing shell profiles. Oracle JDK requires manual license acceptance. There is no automatic rollback.
