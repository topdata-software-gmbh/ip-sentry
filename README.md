# ip-sentry

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

Real-time Nginx access-log monitor with proactive block-event generation for [Fail2Ban](https://www.fail2ban.org/).

`ip-sentry` tails one or more Nginx access logs, applies configurable blocking logic (rate limiting, GeoIP country blacklist, hostname blacklist/whitelist), and writes synthetic block events to a dedicated log file. Fail2Ban reads that log and executes bans via its existing firewall backend.

---

## Features

- **Real-time log tailing** – follows multiple Nginx access log files simultaneously
- **Rate limiting** – blocks IPs exceeding a configurable requests-per-minute threshold
- **GeoIP country blacklist** – rejects traffic from specified countries (MaxMind GeoLite2)
- **Hostname blacklist/whitelist** – filter by reverse-DNS hostname patterns
- **Fail2Ban bridge** – writes structured block events that Fail2Ban can act on
- **Docker-ready** – ships with a minimal multi-stage Dockerfile

---

## Requirements

- Go 1.21+ (for local builds)
- [MaxMind GeoLite2-City database](https://dev.maxmind.com/geoip/geolite2-free-geolocation-data) (optional but recommended)

---

## Configuration

Default config path: `configs/config.yaml`

```yaml
log_sources:
  - "/var/log/nginx/access.log"
block_log_output: "/var/log/nginx/go_blocks.log"
fail2ban_config_path: "/etc/fail2ban"
geoip_db_path: "/app/data/geoip/GeoLite2-City.mmdb"

thresholds:
  max_requests_per_minute: 100

blacklist:
  countries: ["CN", "RU", "IN"]
  hostnames: [".amazonaws.com", ".compute.internal"]

whitelist:
  hostnames: [".googlebot.com", ".search.msn.com"]
```

All fields can be overridden via environment variables (prefix: `IPSENTRY_`).

---

## Building & Running Locally

```bash
# Build
go build -o ip-sentry .

# Run
./ip-sentry run --config configs/config.yaml
```

Or without building:

```bash
go run . run --config configs/config.yaml
```

### Fetch the GeoIP Database

```bash
bash scripts/fetch-geoip-database-v2.sh
```

---

## Docker

### Build Image

```bash
docker build -t ip-sentry:latest .
```

### Run Container

```bash
docker run --rm \
  -v /var/log/nginx:/var/log/nginx \
  -v $(pwd)/configs:/app/configs:ro \
  -v $(pwd)/data/geoip:/app/data/geoip:ro \
  ip-sentry:latest run --config /app/configs/config.yaml
```

---

## Docker Compose Integration

Add the following service to your `docker-compose.yaml` alongside your Nginx and Fail2Ban services:

```yaml
ip-sentry:
  image: ip-sentry:latest
  container_name: ip-sentry
  restart: unless-stopped
  volumes:
    - ./vol/nginx_logs:/var/log/nginx
    - ../topdata-ip-aggregator/configs:/app/configs:ro
    - ../topdata-ip-aggregator/data/geoip:/app/data/geoip:ro
  command: ["run", "--config", "/app/configs/config.yaml"]
```

---

## Block Event Format

The monitor writes one line per event to `block_log_output`:

```text
<IP> - BLOCK_REQUESTED - Reason:<REASON> Country:<ISO_OR_DASH> Host:<HOST_OR_DASH>
```

Example:

```text
198.51.100.42 - BLOCK_REQUESTED - Reason:RATE_LIMIT_EXCEEDED_100_PER_MIN Country:RU Host:ec2-198-51-100-42.compute.internal
```

Fail2Ban can then be configured to watch this file and ban the IP using its standard actions.

---

## License

MIT — see [LICENSE](LICENSE).
