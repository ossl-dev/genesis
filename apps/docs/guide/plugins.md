# Plugins

YAML builtin `type` entries and full Go/YAML configuration share strict option
validation. Configured IDs are unique; dependencies refer to IDs. Builtins load
from the compiled registry. Custom plugins use Go registration or the portable
executable protocol.

All eleven plugins are described in the [overview](/plugins/overview). Lifecycle
and extension details: [plugin development](/guide/plugin-development),
[lifecycle](/plugins/lifecycle), and [Plugin API](/api/plugin).
