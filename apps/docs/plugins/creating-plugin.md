# Creating a Plugin

See the [Plugin Development guide](/guide/plugin-development) for a typed factory and implementation example, and the [Plugin API](/api/plugin) for available methods.

Keep module IDs resolvable by the runtime and IDs unique within a config. Validate options during loading, declare dependencies explicitly, register system prerequisites before apply, and verify the requested state before reporting success.
