#!/bin/bash

# Use with `./release.sh <DESIRED_NEW_VERSION>`

set -e

# Check that we are on the main branch
BRANCH=$(git rev-parse --abbrev-ref HEAD)
echo $BRANCH

if [ $BRANCH != "main" ]; then
    echo "Not on main, aborting"
    exit 1
fi

# Check that the new desired version number was specified correctly
if [ -z "$1" ]; then
    echo "Must specify a desired version number"
    exit 1
elif [[ ! $1 =~ [0-9]+\.[0-9]+\.[0-9]+ ]]; then
    echo "Must use a semantic version, e.g., 3.1.4"
    exit 1
else
    NEW_VERSION=$1
fi

# Check version numbers and print confirmation message
# Get dd-trace-go version number from go.mod file
# Replace version numbers in version.go
# Push new version to GitHub
# Tag new release