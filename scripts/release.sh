#!/bin/bash

# Run from the root directory.
# Use with `./release.sh <DESIRED_NEW_VERSION>`

set -e

# Ensure on main, and pull the latest
BRANCH=$(git rev-parse --abbrev-ref HEAD)
if [ $BRANCH != "main" ]; then
    echo "Not on main, aborting"
    exit 1
else
    echo "Updating main"
    git pull origin main
fi

# Ensure no uncommitted changes
if [ -n "$(git status --porcelain)" ]; then
    echo "Detected uncommitted changes, aborting"
    exit 1
fi

# Check that the new desired version number was specified correctly
if [ -z "$1" ]; then
    echo "Must specify a desired version number"
    exit 1
else
    # Remove 'v' prefix if present
    CLEAN_VERSION=${1#v}

    if [[ ! $CLEAN_VERSION =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
        echo "Must use a semantic version, e.g., 3.1.4"
        exit 1
    else
        NEW_VERSION=$CLEAN_VERSION
    fi
fi

CURRENT_DD_TRACE_VERSION="$(grep "const DDTraceVersion" internal/version/version.go | grep -o -E "[0-9]+\.[0-9]+\.[0-9]+")"
NEW_DD_TRACE_VERSION="$(grep "dd-trace-go.v1" go.mod | grep -o -E "[0-9]+\.[0-9]+\.[0-9]+")"
if [ "$CURRENT_DD_TRACE_VERSION" != "$NEW_DD_TRACE_VERSION" ]; then
    read -p "Confirm updating dd-trace-go version from $CURRENT_DD_TRACE_VERSION to $NEW_DD_TRACE_VERSION (y/n)?" CONT
    if [ "$CONT" != "y" ]; then
        echo "Exiting"
        exit 1
    fi
fi

CURRENT_VERSION="$(grep "const DDLambdaVersion" internal/version/version.go | grep -o -E "[0-9]+\.[0-9]+\.[0-9]+")"
read -p "Ready to update the library version from $CURRENT_VERSION to $NEW_VERSION and release the library (y/n)?" CONT
if [ "$CONT" != "y" ]; then
    echo "Exiting"
    exit 1
fi

# Replace version numbers in version.go
sed -E -i '' "s/(DDLambdaVersion = \")[0-9]+\.[0-9]+\.[0-9]+/\1$NEW_VERSION/g" internal/version/version.go
sed -E -i '' "s/(DDTraceVersion = \")[0-9]+\.[0-9]+\.[0-9]+/\1$NEW_DD_TRACE_VERSION/g" internal/version/version.go

# # Commit change
git commit internal/version/version.go -m "Bump version to ${NEW_VERSION}"
git push origin main

# # Tag new release
git tag "v$NEW_VERSION"
git push origin "refs/tags/v$NEW_VERSION"

echo
echo "Now create a new release with the tag v${NEW_VERSION}"
echo "https://github.com/DataDog/datadog-lambda-go/releases/new?tag=v$NEW_VERSION&title=v$NEW_VERSION"


