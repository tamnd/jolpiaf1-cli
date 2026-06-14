---
title: "Installation"
description: "Install jolpiaf1 from a release, with go install, or from source."
weight: 20
---

## Prebuilt binaries

Every [release](https://github.com/tamnd/jolpiaf1-cli/releases) carries archives for Linux, macOS,
and Windows on amd64 and arm64, plus deb, rpm, and apk packages for Linux.
Download, unpack, put `jolpiaf1` on your `PATH`, done. The `checksums.txt`
on each release is signed with keyless [cosign](https://docs.sigstore.dev/) if
you want to verify before running.

## With Go

```bash
go install github.com/tamnd/jolpiaf1-cli/cmd/jolpiaf1@latest
```

That puts `jolpiaf1` in `$(go env GOPATH)/bin`, which is `~/go/bin` unless
you moved it. Make sure that directory is on your `PATH`.

## From source

```bash
git clone https://github.com/tamnd/jolpiaf1-cli
cd jolpiaf1-cli
make build        # produces ./bin/jolpiaf1
./bin/jolpiaf1 version
```

## Container image

```bash
docker run --rm ghcr.io/tamnd/jolpiaf1:latest --help
```

## Checking the install

```bash
jolpiaf1 version
```

prints the version and exits.
