# Gatherer

Gatherer is a project that caches build dependencies on ephemeral file
systems, such as [drone.io](https://drone.io). It is built with 3
principles in mind.

1. Minimize dependency hosting expense.
2. Easy to deploy.
3. Easy to maintain.

## Installation

    curl gatherer https://drone.io/github.com/justinabrahms/gatherer/files/gatherer > gatherer
    # or on OSX..
    curl gatherer https://drone.io/github.com/justinabrahms/gatherer/files/gatherer_osx > gatherer

    chmod +x gatherer

## Usage

Gatherer works by calculating a hash from a list of files indicated by
`--toHash`. It looks for a bundle matching that hash on Amazon's
S3. If it finds it, it will download and extract the files. If the
hash didn't exist or there was an error during the download, Gatherer
will run your `--buildCommand` which should generate files in
`--packageDirectories` (a comma-separated list of directories). It
will archive these files and store them in S3 for later retrieval.

**NOTE**: The gatherer binary expects you to have 2 environment
variables with your S3 credentials: `AWS_ACCESS_KEY_ID` and
`AWS_SECRET_ACESS_KEY`.

### Node

    gatherer --packageDirectories="node_modules" \
             --toHash="package.json" \
             --buildCommand="npm install ." \
             --bucketName="dependency-artifacts"