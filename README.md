# Flyenv

If you need to work with multiple [Concourse](https://concourse-ci.org/) instances running on different version,
you must have run `fly -t <target> sync` a couple times when you jumping back and forth between those instances.

flyenv could be **the** solution.

It manages the `fly` cli version for you:

```
$ mv flyenv $(command -v fly)
$ fly --version # it downloads the latest fly cli from github when running for the first time
2020/04/10 19:54:06 download cli from: https://github.com/concourse/concourse/releases/download/v6.0.0/fly-6.0.0-darwin-amd64.tgz
6.0.0-flyenv

$ fly -t c571 curl /api/v1/info
2020/04/10 19:56:03 download cli from: https://c571.local/api/v1/cli?arch=amd64&platform=darwin
{"version":"5.7.1","worker_version":"2.2","external_url":"https://c571.local"}

$ fly -t c580 curl /api/v1/info
2020/04/10 19:57:00 download cli from: https://c580.local/api/v1/cli?arch=amd64&platform=darwin
{"version":"5.8.0","worker_version":"2.2","external_url":"https://c580.local"}

$ fly -t c571 --version # "-flyenv" suffix is added to differentiate with fly cli
5.7.1-flyenv

$ fly --version # for commands without a target it fallback to use the cli downloaed initially
6.0.0-flyenv
```

# How it works
When you use the command with the specific target, it fetches the concourse version of the target and determine if the
corresponding fly version is already installed under `$HOME/.flyenv/<version>/fly`. If not it downloads the fly cli
from the target and put it to the directory above.

# Skip SSL Validation

Sometimes you may have Concourse targets that don't have valid certificates. You can tell flyenv to skip SSL validation by setting the `FLYENV_SKIP_SSL` environment variable before running `fly` commands.

# Installation

Download the binary from release.

## Recommended:
override the fly installed with the flyenv binary to have a more transparent experience.

# Install from source

1. clone the repo
1. go mod download
1. go build
