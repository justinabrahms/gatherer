# Gatherer

Gatherer is a project that caches build dependencies on ephemeral file
systems, such as [drone.io](https://drone.io). It is built with 3
principles in mind.

1. Minimize dependency hosting expense.
2. Easy to deploy.
3. Easy to maintain.

## Usage

Gatherer works by calculating a hash from a list of files indicated by
`--toHash`. It looks for a bundle matching that hash on Amazon's
S3. If it finds it, it will download and extract the files. If the
hash didn't exist or there was an error during the download, Gatherer
will run your `--buildCommand` which should generate files in
`--packageDirectories` (a comma-separated list of directories). It
will archive these files and store them in S3 for later retrieval.

### Node

    gatherer --packageDirectories="node_modules" \
             --toHash="package.json" \
             --buildCommand="npm install"
