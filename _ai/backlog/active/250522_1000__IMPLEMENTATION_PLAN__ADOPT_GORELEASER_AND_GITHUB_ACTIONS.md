---
filename: "_ai/backlog/active/250522_1000__IMPLEMENTATION_PLAN__ADOPT_GORELEASER_AND_GITHUB_ACTIONS.md"
title: "Adopt GoReleaser and GitHub Actions for Automated CI/CD"
createdAt: 2025-05-22 10:00
updatedAt: 2025-05-22 10:00
status: draft
priority: high
tags: [golang, goreleaser, github-actions, docker, cicd]
estimatedComplexity: moderate
documentType: IMPLEMENTATION_PLAN
---

# Problem Statement
The current release process for `ip-sentry` is manual, relying on local shell scripts for building and pushing Docker images. This approach lacks transparency, versioning consistency, and multi-platform support (e.g., ARM64 for Raspberry Pi users). There is no automated testing (CI) to ensure code quality on pull requests or pushes to the main branch.

# Executive Summary
This plan implements a professional CI/CD pipeline using **GitHub Actions** and **GoReleaser**. 
1.  **CI Workflow**: Automatically runs Go tests and linting on every push and pull request.
2.  **GoReleaser Integration**: Configures GoReleaser to produce multi-platform binaries (Linux, macOS, Windows) and generate automated GitHub Releases with changelogs.
3.  **Automated Container Registry**: Builds and pushes multi-arch Docker images (AMD64/ARM64) to **GitHub Container Registry (GHCR)** automatically on version tags.
4.  **Documentation**: Updates the README to guide users toward the official automated releases and container images.

# Project Environment
- Project Name: `ip-sentry`
- Language: Go 1.21+
- CLI Framework: Cobra
- Target Registry: GitHub Container Registry (ghcr.io)
- Platforms: Linux (amd64, arm64), Darwin (amd64, arm64), Windows (amd64)

---

## Phase 1: GoReleaser Configuration

We will create the GoReleaser configuration file to define how binaries and Docker images are built.

[NEW FILE] `.goreleaser.yaml`
```yaml
version: 2
project_name: ip-sentry

before:
  hooks:
    - go mod tidy

builds:
  - env:
      - CGO_ENABLED=0
    goos:
      - linux
      - darwin
      - windows
    goarch:
      - amd64
      - arm64
    ldflags:
      - -s -w -X github.com/topdata-software-gmbh/ip-sentry/cmd.version={{.Version}}

archives:
  - format: tar.gz
    # use zip for windows archives
    format_overrides:
      - goos: windows
        format: zip
    name_template: >-
      {{ .ProjectName }}_
      {{- title .Os }}_
      {{- if eq .Arch "amd64" }}x86_64
      {{- else if eq .Arch "386" }}i386
      {{- else }}{{ .Arch }}{{ end }}
      {{- if .Arm }}v{{ .Arm }}{{ end }}

checksum:
  name_template: 'checksums.txt'

snapshot:
  version_template: "{{ incpatch .Version }}-next"

changelog:
  sort: asc
  filters:
    exclude:
      - '^docs:'
      - '^test:'

dockers:
  - image_templates:
      - "ghcr.io/{{ .Env.GITHUB_REPOSITORY_OWNER }}/ip-sentry:{{ .Version }}"
      - "ghcr.io/{{ .Env.GITHUB_REPOSITORY_OWNER }}/ip-sentry:latest"
    dockerfile: Dockerfile.release
    use: buildx
    build_flag_templates:
      - "--label=org.opencontainers.image.title={{ .ProjectName }}"
      - "--label=org.opencontainers.image.description=Real-time Nginx log monitor with fail2ban bridge"
      - "--label=org.opencontainers.image.source={{ .GitURL }}"
      - "--label=org.opencontainers.image.version={{ .Version }}"
      - "--label=org.opencontainers.image.created={{ .Date }}"
      - "--platform=linux/{{ .Arch }}"
```

[NEW FILE] `Dockerfile.release`
```dockerfile
# Minimal runtime image for GoReleaser builds
FROM alpine:latest

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app
RUN mkdir -p /app/configs /app/data/geoip

# GoReleaser copies the binary into the root of the build context
COPY ip-sentry /usr/local/bin/ip-sentry
COPY scripts/ /app/scripts/

# Create a startup preamble
RUN echo "ip-sentry starting - build: $(date -u +'%Y-%m-%dT%H:%M:%SZ')" > /app/preamble.txt

# Entrypoint script
RUN printf '#!/bin/sh\ncat /app/preamble.txt\nexec "$@"\n' > /app/entrypoint.sh && chmod +x /app/entrypoint.sh

ENTRYPOINT ["/app/entrypoint.sh", "/usr/local/bin/ip-sentry"]
CMD ["run", "--config", "/app/configs/config.yaml"]
```

---

## Phase 2: GitHub Actions Workflows

We will create two workflows: one for quality checks and one for automated releases.

[NEW FILE] `.github/workflows/check.yml`
```yaml
name: Quality Check

on:
  push:
    branches: [ main ]
  pull_request:
    branches: [ main ]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      
      - name: Set up Go
        uses: actions/setup-go@v5
        with:
          go-version: '1.21'
          cache: true

      - name: Lint
        uses: golangci/golangci-lint-action@v4
        with:
          version: latest

      - name: Test
        run: go test -v -race ./...
```

[NEW FILE] `.github/workflows/release.yml`
```yaml
name: Release

on:
  push:
    tags:
      - 'v*'

permissions:
  contents: write
  packages: write

jobs:
  goreleaser:
    runs-on: ubuntu-latest
    steps:
      - name: Checkout
        uses: actions/checkout@v4
        with:
          fetch-depth: 0

      - name: Set up Go
        uses: actions/setup-go@v5
        with:
          go-version: '1.21'
          cache: true

      - name: Set up QEMU
        uses: docker/setup-qemu-action@v3

      - name: Set up Docker Buildx
        uses: docker/setup-buildx-action@v3

      - name: Login to GitHub Container Registry
        uses: docker/login-action@v3
        with:
          registry: ghcr.io
          username: ${{ github.actor }}
          password: ${{ secrets.GITHUB_TOKEN }}

      - name: Run GoReleaser
        uses: goreleaser/goreleaser-action@v5
        with:
          distribution: goreleaser
          version: latest
          args: release --clean
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
```

---

## Phase 3: Project Cleanup

Now that we have automated builds, we can remove the manual scripts.

[DELETE] `scripts/build-and-push-docker-image.sh`

---

## Phase 4: Documentation Update

[MODIFY] `README.md`
```markdown
## Installation

### Container Image (Recommended)

Images are automatically published to GitHub Container Registry.

```yaml
services:
  ip-sentry:
    image: ghcr.io/topdata-software-gmbh/ip-sentry:latest
    # ... rest of config
```

### Binary Downloads

Download the pre-compiled binaries for Linux, macOS, and Windows from the [Releases](https://github.com/topdata-software-gmbh/ip-sentry/releases) page.

---

## Development

### CI/CD
This project uses GitHub Actions for CI/CD:
- **Linting & Tests**: Triggered on every Pull Request to `main`.
- **Releases**: Triggered by pushing a git tag (e.g., `git tag v1.0.0 && git push origin v1.0.0`). GoReleaser handles the build and publication.
```

---

## Phase 5: Implementation Report

[NEW FILE] `_ai/backlog/reports/250522_1200__IMPLEMENTATION_REPORT__ADOPT_GORELEASER_AND_GITHUB_ACTIONS.md`

```yaml
---
filename: "_ai/backlog/reports/250522_1200__IMPLEMENTATION_REPORT__ADOPT_GORELEASER_AND_GITHUB_ACTIONS.md"
title: "Report: Adopt GoReleaser and GitHub Actions"
createdAt: 2025-05-22 12:00
updatedAt: 2025-05-22 12:00
planFile: "_ai/backlog/active/250522_1000__IMPLEMENTATION_PLAN__ADOPT_GORELEASER_AND_GITHUB_ACTIONS.md"
project: "ip-sentry"
status: completed
filesCreated: 5
filesModified: 1
filesDeleted: 1
tags: [automation, cicd, goreleaser]
documentType: IMPLEMENTATION_REPORT
---

## Summary
Successfully implemented a modern CI/CD pipeline using GitHub Actions and GoReleaser. The project now supports automated testing, linting, multi-platform binary releases, and multi-arch Docker images hosted on GHCR.

## Files Changed
- **.goreleaser.yaml** [NEW]: Core configuration for builds and releases.
- **Dockerfile.release** [NEW]: Optimized Dockerfile for GoReleaser integration.
- **.github/workflows/check.yml** [NEW]: Continuous Integration (Lint/Test).
- **.github/workflows/release.yml** [NEW]: Continuous Deployment (Release/Push).
- **README.md** [MODIFY]: Updated installation instructions for GHCR and binaries.
- **scripts/build-and-push-docker-image.sh** [DELETE]: Removed manual build script.

## Key Changes
- **Multi-Arch Support**: Added builds for `arm64` (Raspberry Pi/M1 Macs) alongside `amd64`.
- **GHCR Integration**: Container images are now pushed to `ghcr.io` using the native `GITHUB_TOKEN`.
- **Automated Changelog**: Releases now include an automatically generated changelog based on commit history.
- **Stateless Docker**: Switched to `Dockerfile.release` which expects the binary to be pre-built, reducing container image size.

## Technical Decisions
- **Docker Buildx**: Used Buildx in GitHub Actions to enable multi-platform container builds.
- **QEMU**: Included QEMU setup in the release workflow to allow cross-architecture emulation during the Docker build phase.
- **GitHub Registry**: Preferred GHCR over Docker Hub to minimize external credential management and take advantage of GitHub's free tier for public packages.

## Testing Notes
1. Create a tag: `git tag v0.2.0-test`.
2. Push tag: `git push origin v0.2.0-test`.
3. Monitor the **Actions** tab in GitHub to verify the release workflow completes successfully.
4. Verify the new release appears on the GitHub Release page with attached binaries.
```

