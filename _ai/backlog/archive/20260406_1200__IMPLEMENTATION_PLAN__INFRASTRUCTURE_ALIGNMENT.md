---
filename: "_ai/backlog/active/20260406_1200__IMPLEMENTATION_PLAN__INFRASTRUCTURE_ALIGNMENT.md"
title: "Align Aggregator with Nginx-Proxy Infrastructure"
createdAt: 2026-04-06 12:00
updatedAt: 2026-04-06 12:30
status: completed
priority: high
tags: [golang, infrastructure, fail2ban, nginx-proxy]
estimatedComplexity: moderate
documentType: IMPLEMENTATION_PLAN
---

# Problem Statement
The Go Aggregator had a redundant local vol directory and incomplete path mapping to the external topdata-nginx-proxy-v2 infrastructure.

# Implemented Steps
1. Removed redundant local vol assets from the repository.
2. Updated runtime configuration to use /srv/topdata-nginx-proxy-v2/vol paths.
3. Added fail2ban_config_path to the config model.
4. Improved monitor startup telemetry and path validation for log sources, fail2ban config path, and bridge log output.
5. Updated README configuration examples to match the production mapping.

# Outcome
The aggregator now points directly to the proxy stack directories and provides clearer startup diagnostics.
