# Flyenv

If you need to work with multiple [Concourse](https://concourse-ci.org/) instances running on different version, 
you must have run `fly -t <target> sync` a couple times when you jumping back and forth between those instances.

flyenv could be **the** solution.

It manages the `fly` cli version for you:

```
$ ./flyenv -t c571 curl /api/v1/info
2020/04/10 19:37:48 download cli from: https://c571.local/api/v1/cli?arch=amd64&platform=darwin
{"version":"5.7.1","worker_version":"2.2","external_url":"https://c571.local"}
$ ./flyenv -t c580 curl /api/v1/info
2020/04/10 19:39:13 download cli from: https://c580.local/api/v1/cli?arch=amd64&platform=darwin
{"version":"5.8.0","worker_version":"2.2","external_url":"https://c580.local"}
```

# How it works
When you use the command with the specific target, it fetches the concourse version of the target and determine if the
corresponding fly version is already installed under `$HOME/.flyenv/<version>/fly`. If not it downloads the fly cli
from the target and put it to the directory above.

# Installation

Download the binary from release.
Recommended: override the fly installed with the flyenv binary to have a more transparent experience. 

# Install from source
 
1. clone the repo
1. go mod download
1. go build
