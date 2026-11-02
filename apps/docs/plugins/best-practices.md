# Plugin Practices

- Validate and default options before touching the host.
- Detect matching installations before registering expensive prerequisites.
- Deduplicate system tasks with exact IDs and explicit dependencies.
- Install runtime-specific global packages after the runtime is ready.
- Pass runtime cwd/env and caller context to every operation. Use argument arrays;
  pass user values positionally when intentionally invoking a shell.
- Stream downloads, verify published checksums, stage and verify archives before
  promotion, and preserve recovery backups when rollback fails.
- Bound goroutines, output, downloads, and deadlines. Wait for workers and child
  processes before returning. Avoid process-wide environment mutation.
- Return actionable errors and truthful change status. Never swallow failed
  package operations or claim unimplemented cloud/restore actions succeeded.
- Test real temporary-file behavior and native platforms alongside command mocks.
