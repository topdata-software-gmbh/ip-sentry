---
filename: "_ai/backlog/active/260406_1245__IMPLEMENTATION_PLAN__OPEN_SOURCE_DOCKER_INTEGRATION.md"
title: "Open-Source Release, Dockerization, and Compose Integration for IP Aggregator"
createdAt: 2026-04-06 12:45
createdBy: GitHub Copilot [Gemini 3.1 Pro (Preview)]
updatedAt: 2026-04-06 12:45
updatedBy: GitHub Copilot [Gemini 3.1 Pro (Preview)]
status: completed
priority: high
tags: [open-source, docker, docker-compose, refactoring]
project: ip-sentry
estimatedComplexity: moderate
documentType: IMPLEMENTATION_PLAN
---

# Plan: Open-Source Release, Dockerization, and Compose Integration

## Problem Statement
The current `topdata-ip-aggregator` project needs to be prepared for an open-source release. This involves finding a better, generic name (proposed: `ip-sentry`), finalizing an open-source structure (LICENSE, generalized `README.md`), wrapping the application in a portable `Dockerfile`, and linking it directly into the `topdata-nginx-proxy` docker-compose stack.

## Implementation Notes
- **Context:** The GO app parses Nginx logs and monitors IP behavior. It relies on a local GeoIP database (`GeoLite2-City.mmdb`) and configuration. 
- **Proposed Name:** `ip-sentry` (The plan uses this name, but it is a placeholder that can be adjusted in the refactoring phase).
- **Paths:**
  - Workspace Root: `/home/marc/workspaces/ip-block/`
  - Go Project Dir: `/home/marc/workspaces/ip-block/topdata-ip-aggregator/`
  - Proxy Compose Dir: `/home/marc/workspaces/ip-block/topdata-nginx-proxy/`
- **Commands Needed for Testing:**
  - `docker build -t ip-sentry:latest .`
  - `docker compose up -d` in the proxy directory to verify integration.
- SOLID principles apply here primarily in how cleanly the environment variables inject into the Docker container vs the original hardcoded configs.

---

## Phase 1: Project Renaming and Clean-up
**Objective**: Rename `topdata-ip-aggregator` internal references to a generic open-source friendly name like `ip-sentry`.

**Tasks**:
1. Scan for `topdata-ip-aggregator` in `go.mod`, import statements across all `/internal/` and `/cmd/` packages, and configuration scripts. 
2. Replace it with `github.com/USERNAME/ip-sentry` (or a generic placeholder for the open-source repo).
3. Validate that the application compiles successfully after the package rename. (`go build -v ./...`)

**Deliverables**:
- Updated `go.mod`
- All Go files modified to reflect new imports.

---

## Phase 2: Open-Source Documentation and Structuring
**Objective**: Prepare public-facing documentation and add a license.

**Tasks**:
1. Add an MIT `LICENSE` file.
2. Ensure the `README.md` is generalized for open-source users. It should cover:
   - What the project does (Parsing Nginx logs, integrating with Fail2Ban, GeoIP lookups).
   - How to build and run the binary locally.
   - How to run it via Docker / Docker Compose.
   - Example configuration.

**Deliverables**:
- `[NEW FILE] topdata-ip-aggregator/LICENSE`
- `[MODIFY] topdata-ip-aggregator/README.md`

---

## Phase 3: Dockerization
**Objective**: Create a lean `Dockerfile` to build and run the Go application.

**Tasks**:
1. Create a `.dockerignore` to keep the context small.
2. Develop a multi-stage `Dockerfile` based on `golang:1.VER-alpine`.
   - **Stage 1**: Build the Go binary using `go build`.
   - **Stage 2**: Use a runtime image (e.g., `alpine:latest`).
   - The final stage should set up the necessary directory structures `/app`, `/app/configs`, `/app/data/geoip/`.
   - Consider the GeoIP fetching script: It might need to run at build time, or the host can mount it. We will assume the script is included so users can pull the DB if not volume-mounted.
3. Configure `ENTRYPOINT` to `["./ip-sentry"]`.

**Deliverables**:
- `[NEW FILE] topdata-ip-aggregator/Dockerfile`
- `[NEW FILE] topdata-ip-aggregator/.dockerignore`

---

## Phase 4: Integration with `topdata-nginx-proxy` Compose Stack
**Objective**: Add the newly containerized service to the existing `docker-compose.yaml` proxy setup.

**Tasks**:
1. Add `ip-sentry` as a service in `/home/marc/workspaces/ip-block/topdata-nginx-proxy/docker-compose.yaml`.
2. Map necessary Docker volumes:
   - `- ./vol/nginx_logs:/var/log/nginx:ro` for Nginx log tailing.
   - `- ../topdata-ip-aggregator/configs:/app/configs:ro` (to mount the config.yaml for now, or adapt it to purely ENV vars).
   - `- ../topdata-ip-aggregator/data/geoip:/app/data/geoip:ro` (to provide the GeoIP database).
3. Ensure it resides in the same `nginx-proxy-net` network if needed.

**Deliverables**:
- `[MODIFY] topdata-nginx-proxy/docker-compose.yaml`

---

## Phase 5: Verification & Implementation Report
**Objective**: Verify the full stack starts correctly and document the outcome.

**Tasks**:
1. Run `docker compose up -d` in the `topdata-nginx-proxy` folder.
2. Check `docker logs nginx-proxy` and `docker logs ip-sentry` to ensure logs are processed without pathing errors.
3. Generate the final Implementation Report.

**Deliverables**:
- `[NEW FILE] _ai/backlog/reports/260406_1245__IMPLEMENTATION_REPORT__OPEN_SOURCE_DOCKER_INTEGRATION.md`
