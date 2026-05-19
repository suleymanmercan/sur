# Release Guide

This document explains how to cut a new `sur` release.

## Prerequisites

- You must have push access to the `suleymanmercan/sur` repository.
- All changes must be merged to `main` and CI must be green before tagging.

## Steps

### 1. Verify CI is green

Go to the [Actions tab](https://github.com/suleymanmercan/sur/actions) and confirm:

- The latest `main` commit has a passing **CI** run (test + lint + security).

### 2. Choose a version number

`sur` follows [Semantic Versioning](https://semver.org/):

| Change type | Example |
|-------------|---------|
| Bug fix, small improvement | `v0.1.1` → `v0.1.2` |
| New command or feature | `v0.1.x` → `v0.2.0` |
| Breaking change | `v0.x.x` → `v1.0.0` |

### 3. Tag and push

```sh
# Replace with the actual version
VERSION=v0.2.0

git tag -a "$VERSION" -m "release $VERSION"
git push origin "$VERSION"
```

### 4. Watch the Release workflow

Go to **Actions → Release**. It will:

1. Build `sur-linux-amd64` and `sur-linux-arm64`.
2. Generate `*.sha256` checksum files.
3. Create a GitHub Release and upload all four assets.

The release is done when the workflow turns green.

### 5. Verify install.sh works

From a clean Linux machine (or a Docker container):

```sh
curl -fsSL https://raw.githubusercontent.com/suleymanmercan/sur/main/install.sh | sudo bash
sur --version
```

Expected output: the version you just tagged.

### 6. Verify update works

```sh
curl -fsSL https://raw.githubusercontent.com/suleymanmercan/sur/main/install.sh | sudo bash -s -- --update
sur --version
```

State directories (`/var/lib/sur`) must be untouched after update.

## Hotfix releases

If a critical bug is found after a release:

1. Branch off the tag: `git checkout -b hotfix/v0.2.1 v0.2.0`
2. Apply the fix and commit.
3. Merge to `main`.
4. Tag as `v0.2.1` and follow the steps above.

## Release assets checklist

After the Release workflow finishes, confirm these four files are attached to the GitHub Release:

- [ ] `sur-linux-amd64`
- [ ] `sur-linux-amd64.sha256`
- [ ] `sur-linux-arm64`
- [ ] `sur-linux-arm64.sha256`
