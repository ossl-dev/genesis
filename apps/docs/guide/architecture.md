# Architecture

Genesis turns a project config into host provisioning operations. Its aim is repeatable team onboarding; it does not currently isolate projects or make host installation transactional.

## Packages

- `packages/core`: config parsing/validation, plugin loading and lifecycle, system tasks, environment apply, parallel execution, shell/filesystem utilities.
- `packages/plugins`: eleven built-in plugin factories and implementations.
- `apps/cli`: command parsing, config preparation, output, and process exit handling.
- `apps/docs`: VitePress documentation.

## Local apply

1. Load the selected config; normalize YAML entries, expand environment references, and validate the structure.
2. Import plugins, validate/default options, and sort dependencies. Invalid graphs fail before provisioning.
3. Merge configured environment variables into a copy of the inherited environment.
4. Run before scripts.
5. Register and execute deduplicated system prerequisites. Missing dependencies or cycles fail before task execution; failed prerequisites stop plugin apply.
6. Run plugin pre/apply/post hooks in dependency order. Returned failures block dependents; thrown failures stop execution.
7. After successful plugins, clone/check repositories, then run after scripts.

The CLI reports failures with nonzero exits. Existing repositories are checked for root, origin, and requested branch; they are not pulled, reset, or switched. Scripts always run and must be safe to repeat.

## Execution

The CLI executes plugins sequentially. The [parallel core API](/api/execution) retains real plugin nodes, groups dependency layers, and uses bounded workers. It serializes overlapping configured paths/ports but cannot infer all package-manager or shell-profile conflicts.

## Inspection and unfinished systems

`diff` detects plugins; it is not a full desired/current-state diff. `doctor` detects and validates configured plugins, rather than performing a config-free host inventory.

`EnvironmentCacheManager` remains a prototype: restore methods do not restore files or network state, and its local persistence is not a remote sync service. CLI apply does not populate a usable cache. Cloud commands have no backend integration. The roadmap records these as unfinished.

See [Core API](/api/core), [Configuration](/guide/configuration), and [Lifecycle](/plugins/lifecycle).
