# Genesis

Development setup from a config. Version your team's tools, repositories,
environment variables, and setup scripts together.

- [Get started](/guide/getting-started): build one binary, preview, apply, verify.
- [Platform support](/guide/platform-support): supported installers and known limits.
- [Plugin reference](/plugins/overview): eleven builtins with strict options.
- [Go API](/api/core): lifecycle, shared prerequisites, bounded parallel execution.

Genesis is an early-stage local provisioning tool written in Go. YAML and Go
configurations are validated before provisioning. Shared tasks deduplicate host
prerequisites; dry runs export JSON plans; failed apply and validation exit nonzero.

Cloud environments, cache restoration, and system-wide rollback remain unfinished.
Documentation is built and served by Go with no browser scripting. Search is
available when serving; static exports provide navigation and browser find.
