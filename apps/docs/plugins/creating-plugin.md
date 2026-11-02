# Creating a Plugin

Start with [plugin development](/guide/plugin-development). Use a compiled
`core.Plugin` or an executable implementing protocol version 1. Validate options,
declare dependencies, detect existing tools, and use shared tasks/installers.

Keep commands contextual, cancellation-aware, and safe for reapply. Test missing
tools, failed commands, unsupported platforms, partial installs, dependency
failures, and fresh-context rediscovery. Reference: [Plugin API](/api/plugin).
