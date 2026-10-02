# envbuckets documentation

envbuckets keeps local `.env` values in named buckets and selects a bucket
for each Git branch. Start with the [setup guide](getting-started.md), then
use the guides below as your project grows.

| Guide                                           | What it covers                                      |
| ----------------------------------------------- | --------------------------------------------------- |
| [Getting started](getting-started.md)           | Install, initialize, and set up a fresh clone       |
| [Configuration and rules](configuration.md)     | Shared config, patterns, priority, and rule editing |
| [Buckets](buckets.md)                           | Create, inspect, and safely remove local values     |
| [Branch switching](branch-switching.md)         | Checkout hook, manual switches, and local pins      |
| [Scopes and monorepos](scopes.md)               | Independent `.env` files in one repository          |
| [Status and recovery](status-and-recovery.md)   | Diagnose and repair an unready checkout             |
| [Uninstall and safety](uninstall-and-safety.md) | Data protection and deactivation                    |
| [Command reference](command-reference.md)       | Every command, flag, and exit code                  |

The [README](../README.md) is a quick overview. These pages describe the
current CLI in more detail. Run `envbuckets help` or a command's `--help`
for its built-in usage text.
