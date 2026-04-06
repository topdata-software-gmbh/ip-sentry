# topdata-ip-aggregator

Real-time Nginx access-log monitoring with proactive block-event generation for fail2ban.

## Real-time Aggregator (Go)

This component tails one or more Nginx access logs, applies blocking logic (rate threshold, country blacklist, hostname blacklist/whitelist), and writes synthetic block events to a dedicated log file.

Fail2ban then reads that log and executes bans through its existing firewall backend.

## Configuration

Default config path: `configs/config.yaml`

```yaml
log_sources:
  - "/var/log/nginx/access.log"
block_log_output: "/var/log/nginx/go_blocks.log"
geoip_db_path: "./data/geoip/GeoLite2-City.mmdb"
```

## Running the Monitor

```bash
go run . run --config configs/config.yaml
```

Or after building:

```bash
./topdata-ip-aggregator run --config configs/config.yaml
```

## Block Event Format

The monitor writes one line per event:

```text
<IP> - BLOCK_REQUESTED - Reason:<REASON> Country:<ISO_OR_DASH> Host:<HOST_OR_DASH>
```

Example:

```text
198.51.100.42 - BLOCK_REQUESTED - Reason:RATE_LIMIT_EXCEEDED_100_PER_MIN Country:RU Host:ec2-198-51-100-42.compute.internal
```
