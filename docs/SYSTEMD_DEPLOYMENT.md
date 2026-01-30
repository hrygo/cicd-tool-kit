# Systemd Deployment Guide

This guide describes how to deploy the CICD Toolkit as a persistent service using systemd user services, following the production best practices from [BEST_PRACTICE_CLI_AGENT.md](./BEST_PRACTICE_CLI_AGENT.md).

## Overview

Using systemd user services provides:
- **Automatic process restart** on failure
- **Resource limits** via cgroups
- **Multi-user isolation** without root privileges
- **Lingering mode** for offline operation

## Quick Start

### 1. Install the Binary

```bash
# Install to /usr/local/bin
sudo cp bin/cicd-runner /usr/local/bin/
sudo chmod +x /usr/local/bin/cicd-runner
```

### 2. Create Systemd User Unit

Create the file `~/.config/systemd/user/cicd-runner.service`:

```ini
[Unit]
Description=CICD AI Toolkit Runner
After=network.target

[Service]
Type=simple
Restart=always
RestartSec=5

# Resource limits (cgroups)
CPUQuota=50%
MemoryMax=1G

# Environment
Environment="PATH=/usr/local/bin:/usr/bin:/bin"
Environment="CLAUDE_CONFIG_DIR=%h/.config/cicd-toolkit"

# ExecStart
ExecStart=/usr/local/bin/cicd-runner daemon --config %h/.config/cicd-toolkit/config.yaml

# Logging
StandardOutput=journal
StandardError=journal
SyslogIdentifier=cicd-runner

[Install]
WantedBy=default.target
```

### 3. Enable and Start

```bash
# Reload systemd
systemctl --user daemon-reload

# Enable lingering (keeps service running after logout)
loginctl enable-linger $USER

# Enable and start service
systemctl --user enable cicd-runner.service
systemctl --user start cicd-runner.service
```

### 4. Check Status

```bash
# View service status
systemctl --user status cicd-runner.service

# View logs
journalctl --user -u cicd-runner.service -f
```

## Multi-Session Agent Template

For running multiple isolated agents (e.g., per-project), use a **template unit**:

Create `~/.config/systemd/user/cicd-agent@.service`:

```ini
[Unit]
Description=CICD Agent for %i
After=network.target

[Service]
Type=simple
Restart=always
RestartSec=3

# Resource limits per instance
CPUQuota=25%
MemoryMax=512M

# Session-specific config
Environment="SESSION_ID=%i"
Environment="CLAUDE_CONFIG_DIR=%h/.config/cicd-toolkit/%i"

ExecStart=/usr/local/bin/cicd-runner agent \
    --session-id %i \
    --config %h/.config/cicd-toolkit/config.yaml

StandardOutput=journal
StandardError=journal
SyslogIdentifier=cicd-agent-%i

[Install]
WantedBy=default.target
```

### Starting Instances

```bash
# Start an agent for project "myapp"
systemctl --user start cicd-agent@myapp.service

# Start multiple agents
systemctl --user start cicd-agent@project-a.service
systemctl --user start cicd-agent@project-b.service

# List all running agents
systemctl --user list-units 'cicd-agent@*'
```

## Directory Structure

Following the FHS (Filesystem Hierarchy Standard):

```
/opt/cicd-toolkit/          # Installation directory
├── bin/                    # Binaries
│   └── cicd-runner
├── skills/                 # Built-in skills
└── configs/                # Default configs

/var/lib/cicd-toolkit/      # Runtime data (system-wide)
└── users/
    └── <user_id>/
        ├── sessions/       # Session files
        └── sockets/        # Unix domain sockets

~/.config/cicd-toolkit/     # User configuration
├── config.yaml             # Main config
└── sessions/               # User session cache
```

## Resource Limits

Configure appropriate limits based on your workload:

```ini
[Service]
# CPU limits
CPUQuota=50%               # Max 50% of one CPU core
CPUWeight=100              # Relative weight (1-10000)

# Memory limits
MemoryMax=1G               # Max memory usage
MemorySwapMax=256M         # Max swap usage

# File descriptor limits
LimitNOFILE=65536          # Max open files

# Process limits
LimitNPROC=4096            # Max processes
```

## Security Considerations

### Running as Non-Root User

User services run without root privileges. For additional security:

```ini
[Service]
# Protect system directories
ProtectSystem=strict
ProtectHome=read-only

# Write paths
ReadWritePaths=/var/lib/cicd-toolkit
ReadWritePaths=/tmp

# Network restrictions
# PrivateNetwork=true      # Uncomment to disable network
```

### Environment Variables

Store sensitive values in `~/.config/environment.d/cicd-toolkit.conf`:

```bash
# API keys (never commit to git)
ANTHROPIC_API_KEY=sk-ant-xxxxx
GITHUB_TOKEN=ghp_xxxxx

# Custom paths
XDG_CACHE_HOME=/home/user/.cache
XDG_DATA_HOME=/home/user/.local/share
```

## Monitoring and Logging

### Journal Logs

```bash
# Follow logs in real-time
journalctl --user -u cicd-runner.service -f

# View last 100 lines
journalctl --user -u cicd-runner.service -n 100

# View logs since boot
journalctl --user -u cicd-runner.service --since today

# Filter by log level
journalctl --user -u cicd-runner.service -f -g "ERROR"
```

### Metrics Integration

Use `systemd-exporter` to expose metrics to Prometheus:

```bash
# Install systemd-exporter
go install github.com/prometheus-community/exporter/systemd_exporter@latest

# Run exporter
systemd_exporter --web.listen-address=:9553
```

## Troubleshooting

### Service Fails to Start

```bash
# Check status for error details
systemctl --user status cicd-runner.service

# View recent logs
journalctl --user -u cicd-runner.service -n 50 --no-pager
```

### Permission Issues

```bash
# Verify binary is executable
ls -l /usr/local/bin/cicd-runner

# Check file ownership
ls -la ~/.config/cicd-toolkit/
```

### Lingering Not Working

```bash
# Verify lingering is enabled
loginctl show-user $USER | grep Linger

# Enable if needed
sudo loginctl enable-linger $USER
```

## Advanced: Socket Activation

For on-demand service startup, use socket activation:

Create `~/.config/systemd/user/cicd-runner.socket`:

```ini
[Unit]
Description=CICD Runner Socket

[Socket]
ListenStream=/run/user/%U/cicd-runner.sock
SocketMode=0600

[Install]
WantedBy=sockets.target
```

Create `~/.config/systemd/user/cicd-runner@.service`:

```ini
[Unit]
Description=CICD Runner (socket-activated)
Requires=cicd-runner.socket

[Service]
ExecStart=/usr/local/bin/cicd-runner --socket-activation
StandardInput=socket
```

Enable:

```bash
systemctl --user enable --now cicd-runner.socket
```

## Integration with CI/CD

### GitHub Actions

```yaml
name: AI Review
on: [pull_request]

jobs:
  review:
    runs-on: self-hosted  # Use runner with systemd service
    steps:
      - uses: actions/checkout@v4
      - run: |
          # Trigger review via systemd service
          curl --unix-socket /run/user/$UID/cicd-runner.sock \
            -X POST /review \
            -H "Content-Type: application/json" \
            -d '{"pr": "${{ github.event.number }}"}'
```

## References

- [systemd.service(5)](https://www.freedesktop.org/software/systemd/man/systemd.service.html)
- [systemd.resource-control(5)](https://www.freedesktop.org/software/systemd/man/systemd.resource-control.html)
- [loginctl(1)](https://www.freedesktop.org/software/systemd/man/loginctl.html)
- [BEST_PRACTICE_CLI_AGENT.md](./BEST_PRACTICE_CLI_AGENT.md)
