# Choosing an Environment Workflow

Genesis's current role is local host provisioning from a version-controlled project config. Evaluate it against your team's actual requirements rather than assuming one config provides isolation or complete reproducibility.

| Requirement | Genesis status |
| --- | --- |
| Tools, repositories, and scripts in one config | Implemented with plugin/platform limits |
| Preview before provisioning | Dry-run action plan; config/module loading still executes code |
| Verify requested tools | Plugin detection and validation |
| Isolated project environments | Not implemented |
| Atomic rollback | Not implemented |
| Exact reproducible package versions on every OS | Depends on installer; several pinning paths incomplete |
| Remote team environment service | Not implemented |
| Fresh-system standalone distribution | Not shipped |

Genesis can coexist with container-based services or other environment tooling through explicit setup scripts. There are no measured performance comparisons in this repository. Choose an alternative workflow if your project requires capabilities Genesis has not yet implemented.
