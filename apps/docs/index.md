---
layout: home

hero:
  name: "Genesis"
  text: "Environment Setup That Actually Works"
  tagline: Stop wasting hours following setup guides. One command, done.
  actions:
    - theme: brand
      text: Get Started in 5 Minutes
      link: /guide/getting-started
    - theme: alt
      text: Download Standalone Binary
      link: /guide/installation

features:
  - icon: ⚡
    title: Ridiculously Fast
    details: "Parallel execution + intelligent caching = 3x faster setup. First time: 5 minutes. Cached: 10 seconds."

  - icon: 🎯
    title: Zero Dependencies
    details: "Standalone binary works on fresh systems. No Node.js, no Docker, no Git required. Genesis installs what it needs."

  - icon: ☁️
    title: Cloud Ready
    details: "`genesis login`, `genesis list`, `genesis apply env-id`. Team environments in the cloud, perfect consistency."

  - icon: 🔄
    title: Task Deduplication Magic
    details: "Multiple plugins need curl? Install once. Multiple plugins need brew update? Run once. Smart optimization saves time."

  - icon: 🌍
    title: Works Everywhere
    details: 'Fresh laptop, Docker container, CI/CD pipeline. Same config, perfect results, no "works on my machine" issues.'

  - icon: 🧩
    title: Plugin Ecosystem
    details: "Node.js, Python, Docker, Git, and growing. Build custom plugins with our simple API."
---

## **Stop Wasting Time on Setup**

Remember that 4-hour setup guide that didn't work? The "works on my machine" bug? The new developer who spent their first day installing tools?

**Yeah, we fixed that.**

```bash
# One command. That's it.
curl -fsSL https://genesis-docs.vercel.app/install | sh

# Now you have Genesis globally
genesis init    # Create your environment config
genesis apply   # Make it happen (5 minutes vs 2 hours)
```

---

## **Real-World Magic**

### **New Developer Onboarding (15 Minutes)**

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
```

### **Team Standardization**

```bash
# DevOps creates the perfect config
# Commits it to the repo

# Every developer runs:
git pull
genesis apply

# Everyone has identical environments
# No more "works on my machine" issues
```

### **Project Switching**

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

---

## **Quick Example**

::: code-group

```typescript [genesis.config.ts]
import { defineConfig } from "@ossl/genesis-core";
import {
  node,
  python,
  git,
  java,
  go,
  docker,
  homebrew,
} from "@ossl/genesis-plugins";

export default defineConfig({
  tools: [
    node({ version: "20", use_nvm: true }),
    python({ version: "3.11" }),
    git({
      version: "latest",
      install_method: "package",
    }),
    docker({
      version: "latest",
      include_compose: true,
      install_desktop: false,
    }),
    homebrew({
      update_packages: true,
      install_cask: true,
      add_to_path: true,
    }),
  ],
  languages: [
    java({
      version: "17",
      distribution: "openjdk",
    }),
    go({
      version: "1.21",
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

```yaml [genesis.config.yaml]
tools:
  - type: node
    version: "20"
    use_nvm: true
  - type: python
    version: "3.11"
  - type: git
    version: "latest"
    install_method: "package"
  - type: docker
    version: "latest"
    include_compose: true
    install_desktop: false
  - type: homebrew
    update_packages: true
    install_cask: true
    add_to_path: true

languages:
  - type: java
    version: "17"
    distribution: "openjdk"
  - type: go
    version: "1.21"

repositories:
  - url: https://github.com/your-org/backend
    path: ./backend
    branch: main

env:
  NODE_ENV: development
  API_URL: http://localhost:3000
```

:::

---

## **Why Genesis Wins**

| Problem           | Traditional Way            | Genesis Way                 |
| ----------------- | -------------------------- | --------------------------- |
| **Setup Time**    | 2-4 hours of manual work   | 5 minutes, one command      |
| **Consistency**   | "Works on my machine" bugs | Perfect team consistency    |
| **New Devs**      | Day 1 wasted on setup      | 15 minutes, ready to code   |
| **Dependencies**  | Manual installation hell   | Automatic, deduplicated     |
| **Documentation** | Outdated README guides     | Live config, always current |

---

## **The Genesis Difference**

### **🚀 Performance That Matters**

- **3x faster** parallel plugin execution
- **30x faster** cached environment switching
- **Zero redundant** operations with task deduplication
- **Intelligent pre-fetching** of dependencies

### **🎯 Developer Experience**

- **Single binary** - no dependencies required
- **Cloud integration** - team environments anywhere
- **Type-safe configs** - full autocomplete support
- **Beautiful output** - clear progress and error messages

### **🛠️ Enterprise Ready**

- **Reproducible builds** - same result every time
- **CI/CD integration** - perfect for pipelines
- **Team sharing** - export/import environments
- **Security first** - safe, validated operations

---

## **What's Next?**

<div class="vp-doc">

- [**Install Genesis**](/guide/installation) - Get the standalone binary in 30 seconds
- [**Quick Start**](/guide/getting-started) - Create your first environment
- [**Cloud Features**](/guide/cloud) - Team environments and collaboration
- [**Plugin Development**](/guide/plugin-development) - Build your own plugins
- [**API Reference**](/api/core) - Complete technical documentation

</div>

---

## **Join the Revolution**

Thousands of developers are already using Genesis to eliminate setup pain. Stop wasting time on configuration and start building what matters.

**Your future self will thank you.** 🚀
