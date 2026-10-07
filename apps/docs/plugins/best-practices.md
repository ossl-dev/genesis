# Plugin Practices

- Validate/default options at load time through `parseOptions`; do not discover malformed config after changing the machine.
- Detect matching installations before registering costly package-manager work.
- Use the task registry for shared prerequisites. IDs/dependencies are exact; register update tasks when install tasks depend on them.
- Install runtime-specific packages after that runtime is ready.
- Pass runtime `cwd` and `env` to every child command. Use argument arrays; for intentional shell logic, pass user values as positional parameters.
- Treat every nonzero command result as a failure unless the operation explicitly allows it. Verify executable/version/daemon state where appropriate.
- Make hooks and apply operations safe to repeat. Record changes accurately, and document platform limitations.
- Test meaningful failures and ordering. Mock system installers; use real temporary directories for file operations. Do not treat mock success as platform certification.

[Lifecycle](/plugins/lifecycle) documents dependency failure behavior. There is no automatic rollback.
