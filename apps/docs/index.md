---
layout: home
hero:
  name: Genesis
  text: Development setup from a config
  tagline: Version your team's tools, repositories, and setup scripts together.
  actions:
    - theme: brand
      text: Run from source
      link: /guide/getting-started
    - theme: alt
      text: Current support
      link: /guide/platform-support
features:
  - title: Declarative configuration
    details: YAML and TypeScript configs, validated plugin options, and environment references.
  - title: Shared prerequisites
    details: System tasks are deduplicated across plugins and ordered by dependencies.
  - title: Preview and verify
    details: Dry-run plans, JSON export, and nonzero exits for failed apply or validation.
  - title: Plugin lifecycle
    details: Seven built-in plugins plus dependencies and hooks for custom implementations.
---

Genesis is an early-stage local provisioning tool. Cloud environments, cache restore, rollback, and distributed standalone binaries are unfinished. Platform support varies by plugin; see the [plugin reference](/plugins/overview).
