# Minecraft Server Manager

A command-line tool for managing multiple Minecraft servers on a single machine.

**MSM v0.12.0** is a complete rewrite in Go, replacing the original bash init script with a modern, type-safe CLI.

## Requirements

### Platform Support

| Platform | Status | Notes |
|----------|--------|-------|
| Linux | Full support | Primary target platform |
| macOS | Full support | Requires GNU screen (`brew install screen`) |
| Windows | WSL only | Native Windows not supported |

MSM requires a Unix-like environment due to its reliance on GNU screen for process management, POSIX filesystem semantics, and `/dev/shm` for RAM disk features.

### Dependencies

| Dependency | Required | Purpose |
|------------|----------|---------|
| Go 1.21+ | Build only | Not needed if using pre-built binary |
| GNU screen | Yes | Console and process management |
| rsync | Optional | rsync backups, RAM disk sync |
| rdiff-backup | Optional | Incremental backups with rotation |

### Minecraft Version Support

MSM supports **Minecraft 1.7.0 and later** (Java Edition), including the new year-based versioning scheme (e.g., `26.1`, `26.2`). Servers running Minecraft 1.6.x or earlier are not supported.

## Quick Start

```bash
# 1. Install MSM (see Installation section for details)
make build && sudo make setup && sudo make install

# 2. Create a server
msm server create survival

# 3. Set up a jar group and download the server jar
msm jargroup create minecraft https://piston-data.mojang.com/v1/objects/.../server.jar
msm jargroup getlatest minecraft

# 4. Link the server to the jar
msm jar survival minecraft

# 5. Start the server
msm start survival

# 6. Attach to console (detach with Ctrl+A, D)
msm console survival
```

## Installation

### Prerequisites

**Linux (Debian/Ubuntu):**
```bash
sudo apt update
sudo apt install golang-go screen rsync
```

**Linux (RHEL/Fedora):**
```bash
sudo dnf install golang screen rsync
```

**macOS:**
```bash
brew install go screen rsync
```

### From Source (Recommended)

```bash
git clone https://github.com/msmhq/msm.git
cd msm
make build
sudo make setup      # Creates minecraft user and /opt/msm directories
sudo make install    # Installs binary, config, and cron job
```

Optionally enable auto-start on boot:
```bash
sudo make systemd-install
```

For multi-user setups where multiple people manage their own servers, see [PERMISSIONS.markdown](PERMISSIONS.markdown)

### Pre-built Binaries

Download from the [releases page](https://github.com/msmhq/msm/releases).

**Trade-off:** Pre-built binaries skip the build step but require manual setup. If you have Go installed, building from source is easier because `make setup` and `make install` handle everything.

```bash
# Download binary (Linux amd64 example)
sudo curl -L https://github.com/msmhq/msm/releases/latest/download/msm-linux-amd64 -o /usr/local/bin/msm
sudo chmod +x /usr/local/bin/msm

# Download config
sudo curl -L https://raw.githubusercontent.com/msmhq/msm/main/msm.conf -o /etc/msm.conf

# Create minecraft user and directories
sudo groupadd minecraft
sudo useradd --system --home-dir /opt/msm --shell /bin/bash --gid minecraft minecraft
sudo mkdir -p /opt/msm/{servers,jars,versioning,archives/{worlds,logs,backups},fabric}
sudo chown -R minecraft:minecraft /opt/msm
sudo chmod 2775 /opt/msm/servers

# Install cron job
sudo msm cron install
```

### Verify Installation

```bash
msm version
msm config
```

## Features

- **Multi-server management** - Run multiple Minecraft servers from one machine
- **Server lifecycle** - Start, stop, restart with configurable delays and player warnings
- **World backups** - Zip, rsync, or rdiff-backup with automatic rotation
- **RAM disk support** - Load worlds into RAM for reduced lag
- **Jar management** - Organize jars into groups, link servers to specific versions
- **Fabric mod loader** - Native support for Fabric with version compatibility checks
- **Player management** - Allowlist, operators, bans (players and IPs)
- **Log rolling** - Automatic log compression and archival
- **Console access** - Direct access to server console via screen

## Configuration

MSM reads its configuration from `/etc/msm.conf`. Override with `--config` flag or `MSM_CONF` environment variable.

### Key Settings

```bash
# User to run servers as
USERNAME="minecraft"

# Storage paths
SERVER_STORAGE_PATH="/opt/msm/servers"
JAR_STORAGE_PATH="/opt/msm/jars"

# Server defaults
DEFAULT_RAM="1024"
DEFAULT_STOP_DELAY="10"

# Cron schedule
CRON_MAINTENANCE_HOUR="5"
CRON_ARCHIVE_RETENTION_DAYS="30"
```

### Per-Server Configuration

Each server can override defaults in `<server>/server.conf`:

```bash
USERNAME="minecraft"
RAM="2048"
JAR_PATH="paper.jar"
STOP_DELAY="30"
```

## Permissions

MSM enforces ownership-based permissions:

- Each server has an owner (the `USERNAME` in `server.conf`)
- You can only manage servers you own, unless you're root
- When you create a server, it's automatically owned by you
- The systemd service runs as root, so it can start all servers on boot

```bash
# Alice creates a server (owned by alice)
alice$ msm server create my-server

# Alice can manage it
alice$ msm start my-server

# Bob cannot
bob$ msm stop my-server
Error: permission denied: server "my-server" is owned by user "alice"
```

See [PERMISSIONS.markdown](PERMISSIONS.markdown) for the full guide including multi-user setup.

## Common Commands

### Server Lifecycle

```bash
msm start                    # Start all servers
msm stop [--now]             # Stop all servers
msm start <server>           # Start specific server
msm stop <server> [--now]    # Stop specific server
msm restart <server>         # Restart server
msm status <server>          # Check if running
msm console <server>         # Attach to console
msm cmd <server> "command"   # Send console command
```

### Backups

```bash
msm worlds backup --all      # Backup all worlds (all servers)
msm worlds backup <server>   # Backup worlds for one server
msm backup <server>          # Full server backup (includes configs)
```

### Player Management

```bash
msm allowlist add <server> <player>
msm allowlist remove <server> <player>
msm op <server> <player>
msm kick <server> <player> [reason]
```

### Jar Management

```bash
msm jargroup list
msm jargroup create <name> <url>
msm jargroup getlatest <name>
msm jar <server> <jargroup>
```

## RAM Disk Support

Load worlds into RAM (`/dev/shm`) for reduced I/O latency:

```bash
# Enable RAM for a world
msm worlds ram survival world

# Start server (auto-creates symlink to RAM)
msm start survival

# Force sync to disk
msm worlds todisk survival
```

A background sync daemon automatically syncs RAM to disk every 10 minutes while servers are running.

**Warning:** In case of unexpected shutdown, you may lose up to 10 minutes of changes.

## Fabric Mod Loader

MSM natively supports [Fabric](https://fabricmc.net/):

```bash
# Enable Fabric
msm fabric on survival

# Start server (auto-downloads Fabric launcher)
msm start survival

# Check status
msm fabric status survival

# List available versions
msm fabric versions 1.21.4
```

When Fabric is enabled, MSM prevents upgrading to Minecraft versions that Fabric doesn't support:

```bash
$ msm jar survival minecraft-26.1
Error: fabric does not yet support minecraft 26.1 - upgrade blocked
  Hint: Use --force to override
```

### Fabric Configuration

Add to `<server>/server.conf`:

```bash
FABRIC_ENABLED="true"
FABRIC_LOADER_VERSION="0.16.10"      # Optional: pin specific loader
FABRIC_INSTALLER_VERSION="1.1.0"     # Optional: pin specific installer
```

## Full Command Reference

See [COMMANDS.markdown](COMMANDS.markdown) for the complete command reference, including all flags, environment variables, and exit codes.

## Shell Completion

```bash
# Bash
msm completion bash > /etc/bash_completion.d/msm

# Zsh
msm completion zsh > "${fpath[1]}/_msm"

# Fish
msm completion fish > ~/.config/fish/completions/msm.fish
```

## Upgrading from Bash MSM

The Go rewrite maintains full compatibility with existing MSM configurations.

### What stays the same:
- Configuration file format (`/etc/msm.conf`)
- Server directory structure
- Per-server `server.conf` files
- World storage layout
- JSON player files (whitelist, ops, bans)

### What's new:
- Single static binary (no bash dependencies)
- Structured logging with `-v/--verbose` flag
- Faster startup and execution
- Automatic RAM sync daemon (no cron required)

### Migration steps:

1. **Stop all servers:**
   ```bash
   msm stop --now
   ```

2. **Backup your configuration:**
   ```bash
   cp /etc/msm.conf /etc/msm.conf.backup
   ```

3. **Clone and build:**
   ```bash
   git clone https://github.com/msmhq/msm.git
   cd msm
   make build
   ```

4. **Remove old bash MSM:**
   ```bash
   sudo make migrate
   ```

5. **Install:**
   ```bash
   sudo make install
   ```

6. **Verify and start:**
   ```bash
   msm config
   msm server list
   msm start
   ```

### Terminology changes:
- `whitelist` commands are now `allowlist`
- `blacklist` commands are now `blocklist`
- The underlying Minecraft files remain unchanged

## Development

```bash
# Run tests
make test

# Run linters
make lint

# Build for all platforms
make build-all
```

## License

MSM is released under the [GPLv3 license](LICENSE.markdown).

## Acknowledgements

This project grew out of [Ahtenus' Minecraft Init Script](https://github.com/Ahtenus/minecraft-init). The Go rewrite maintains compatibility with the original bash MSM configuration format.

## Support

- [Submit an issue](https://github.com/msmhq/msm/issues) for bugs or feature requests
- See the [changelog](CHANGELOG.markdown) for version history
