# Plugin Lifecycle

Load validates options and builds dependency order before provisioning. Builtins
and custom Go plugins use `core.Plugin`; executable plugins use the same lifecycle
through the versioned protocol.

1. `Prepare` runs for all plugins in dependency order, bootstrapping prerequisites
   such as Homebrew before shared package tasks.
2. `RegisterTasks` registers deduplicated system operations.
3. The task registry rejects invalid graphs, then executes shared prerequisites.
4. Each ready plugin runs `PreApply`, `Apply`, and successful `PostApply`.
   Returned failures block dependents. Sequential Go errors stop the run;
   parallel execution records plugin errors and blocks their dependents.

`Detect` and `Validate` inspect tools without provisioning. `RunDiff` currently
uses detection. `ApplyResult` includes `OK`, `DidChange`, and `Details`;
`ValidateResult` uses `OK` and `Message`.

Node global packages and Homebrew global packages run after their runtime is
ready. System prerequisites run before plugin apply and are deduplicated by task
ID. Managed environment mutations are synchronized and remain local to execution.

Default CLI apply is sequential; `--parallel N` uses bounded dependency layers
and serializes overlapping paths/ports. Arbitrary shell/profile conflicts cannot
be inferred; plugins must declare dependencies and avoid global side effects.
