# Core API

Import `github.com/ossl-dev/genesis/packages/core`. All operations return errors explicitly; library calls never set the process exit status.

## Configuration

```go
func LoadConfig(cwd, path string) (Config, error)
func LoadConfigWithEnv(cwd, path string, lookup func(string) (string, bool)) (Config, error)
func LoadConfigWithOptions(cwd, path string, options ConfigLoadOptions) (Config, error)
func NormalizeConfig(raw any) (Config, error)
func ValidateConfig(config Config) error
func CollectInstances(config Config) []Instance
```

An empty path discovers `genesis.config.go`, `.yaml`, `.yml`, then `.json`. Explicit paths resolve against `cwd`. YAML/JSON accept built-in `type` shorthand and repository `path` as an alias for `folder`. Parsing rejects malformed sections, duplicate IDs, missing fields, multiple documents, and files larger than 8 MiB. Config sources must be regular files; symlinks to regular files are allowed. Unix descriptor opens are nonblocking and recheck the file type, preventing named-pipe stalls. `${NAME}` references expand from the supplied lookup or host environment; missing variables report their field path. Shell expressions are never evaluated.

`Config` contains `Tools`, `SDKs`, `Languages`, `Repositories`, `Scripts`, and `Env`. `Instance` has `ID`, `Category`, `Module`, and `Options`. Options remain plugin-specific maps. Repository specs have `URL`, `Folder`, and optional `Branch`. Script specs have `Name`, `Command`, optional `Description`, and `When` (`before` or `after`, default `after`).

Executable Go configurations use the per-call `ConfigLoadOptions.GoConfigLoader` function; `LookupEnv` optionally supplies environment lookup. The CLI captures its cancellation context in this loader. Concurrent calls have independent loaders, without global mutation. Pure YAML/JSON consumers need no compiler. The [configuration guide](/guide/configuration) describes executable configs and their trust boundary.

## Plugins and execution

Built-in loading belongs to `packages/plugins`: `Load(config)` or `LoadWithResolver(config, resolver)` validates options and builds dependency order. `packages/extension` provides executable custom-plugin resolution. Core accepts `[]Node`, allowing callers to construct plugins directly.

```go
func BuildPluginGraph(nodes []Node) ([]Node, error)
func RunDetect(nodes []Node, runtime *Runtime) ([]DetectSummary, error)
func RunDiff(nodes []Node, runtime *Runtime) ([]DetectSummary, error)
func RunValidate(nodes []Node, runtime *Runtime) ([]ValidateSummary, error)
func RunApplyPrerequisites(nodes []Node, runtime *Runtime) error
func RunApply(nodes []Node, runtime *Runtime) ([]ApplySummary, error)
```

Graph validation rejects duplicate IDs, missing dependencies, and cycles before provisioning. `RunDiff` reports detection status rather than a complete package/version diff. `RunApply` runs prerequisites, then each plugin's pre/apply/post hooks. Failed system tasks stop apply. A plugin returning `OK: false` blocks its dependents while independent plugins can continue. A returned hook error stops sequential execution and returns summaries for completed plugins plus a failed summary for the current plugin. The current summary preserves `DidChange` and partial details returned by apply; a post-apply error retains changes from successful apply. Pre-apply errors report no changes because that hook has no change-result field. The wrapped error preserves its cause for `errors.Is`.

## Environment execution

```go
func CreateEnvironmentPlan(config Config, nodes []Node) (Plan, error)
func ApplyEnvironment(config Config, nodes []Node, runtime *Runtime) ([]ApplySummary, error)
func ApplyEnvironmentParallel(config Config, nodes []Node, runtime *Runtime, workers int) ([]ApplySummary, error)
```

Apply copies the runtime environment, merges config values, runs before scripts, applies plugins, checks/clones repositories, then runs after scripts. Failed plugins prevent repository/after-script work. Existing repositories must match their origin and requested branch; Genesis preserves uncommitted changes and never resets or pulls them.

Plans list ordered actions and environment variable names without running plugin hooks or provisioning commands. Environment values are excluded.

See [Plugin API](/api/plugin), [Task Registry](/api/task-registry), [Parallel Execution](/api/execution), and [Utilities](/api/utilities).

## Cache metadata prototype

`NewEnvironmentCacheManager(CacheConfig)` creates an in-memory metadata store. `CacheEnvironment`, `ExportEnvironment`, `ImportEnvironment`, `CreateSnapshot`, `Cleanup`, and `GetStats` retain metadata operations. Metadata retains host platform/architecture and an available Node version. Config checksums use SHA-256; imports are validated and limited to 8 MiB. `CompressMetadata` and `DecompressMetadata` provide bounded gzip transport.

Artifact capture, durable reload, filesystem/network restore, encryption, remote synchronization, tarball export, and Docker export remain unfinished. Unsupported operations return errors; snapshots report `Restorable: false`. CLI apply does not create backups or populate this prototype. Snapshot metadata includes environment values and should be treated as private. Snapshot directory metadata honors a nonempty absolute `GENESIS_HOME`; this override does not change authentication or installer home resolution.
