# Cloud Status

Genesis Cloud is not implemented in this repository.

`genesis login --token <token>` writes token metadata to `~/.genesis/auth.json`. It does not validate the token, open a browser, implement OAuth, or establish a backend connection. Running login without a token prints instructions. `GENESIS_TOKEN` is not consumed by cloud apply. Token files use owner-only permissions on POSIX systems.

`genesis list --cloud` reports that the backend is unavailable and can still list local metadata. `genesis apply <environment-id>` and `genesis apply --cloud` fail explicitly. A local cache entry does not provide working restore.

There are no cloud create/share/sync/ACL/template commands, team permissions, or cloud API endpoints supplied by the codebase. These remain roadmap goals.

Use [local configuration](/guide/configuration) checked into the team's repository for the current workflow.
