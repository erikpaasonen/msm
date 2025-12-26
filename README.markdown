# Minecraft Server Manager

A command-line tool for managing multiple Minecraft servers on a single machine.

**MSM v0.12.0** is a complete rewrite in Go, replacing the original bash init script with a modern, type-safe CLI.

## Installation

### From Source (Recommended)

Requires Go 1.21 or later. This method installs the binary, default config, and cron job automatically.

```bash
git clone https://github.com/msmhq/msm.git
cd msm
make build
sudo make install
```

### Pre-built Binaries

Download the latest release for your platform from the [releases page](https://github.com/msmhq/msm/releases).

```bash
# Linux (amd64)
sudo curl -L https://github.com/msmhq/msm/releases/latest/download/msm-linux-amd64 -o /usr/local/bin/msm
sudo chmod +x /usr/local/bin/msm

# macOS (arm64)
sudo curl -L https://github.com/msmhq/msm/releases/latest/download/msm-darwin-arm64 -o /usr/local/bin/msm
sudo chmod +x /usr/local/bin/msm

# Install default config and cron job
sudo msm cron install
```

### Shell Completion

MSM supports shell completion for bash, zsh, and fish:

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

### Migration steps:

1. **Backup your configuration:**
   ```bash
   cp /etc/msm.conf /etc/msm.conf.backup
   ```

2. **Stop all servers:**
   ```bash
   msm stop --now
   ```

3. **Clone and build:**
   ```bash
   git clone https://github.com/msmhq/msm.git
   cd msm
   make build
   ```

4. **Remove old bash MSM remnants:**
   ```bash
   sudo make migrate
   ```
   This removes `/etc/init.d/msm` and related init.d symlinks.

5. **Install the new binary:**
   ```bash
   sudo make install
   ```

6. **Verify configuration:**
   ```bash
   msm config
   msm server list
   ```

7. **Start your servers:**
   ```bash
   msm start
   ```

### Terminology changes:
- `whitelist` commands are now `allowlist`
- `blacklist` commands are now `blocklist`
- The underlying Minecraft files remain unchanged

## Quick Start

### Create a new server

```bash
# Create server directory structure
msm server create survival

# Set up a jar group and download the server jar
msm jargroup create minecraft https://piston-data.mojang.com/v1/objects/.../server.jar
msm jargroup getlatest minecraft

# Link the server to the jar
msm jar survival minecraft
```

### Basic operations

```bash
# Start/stop/restart
msm start survival
msm stop survival
msm restart survival

# Check status
msm status survival

# Attach to console (detach with Ctrl+A, D)
msm console survival

# Send a command
msm cmd survival "say Hello everyone!"
```

## Features

- **Multi-server management:** Run multiple Minecraft servers from one machine
- **Server lifecycle:** Start, stop, restart with configurable delays and player warnings
- **World backups:** Zip, rsync, or rdiff-backup with rotation
- **RAM disk support:** Load worlds into RAM for reduced lag (see below)
- **Jar management:** Organize jars into groups, auto-download updates
- **Player management:** Allowlist, operators, bans (players and IPs)
- **Log rolling:** Automatic log compression and archival
- **Console access:** Direct access to server console via screen

## RAM Disk Support

MSM can load Minecraft worlds into RAM (`/dev/shm`) for significantly reduced I/O latency. This is especially useful for servers with high player counts or complex redstone.

### How it works

1. **Enable RAM for a world:**
   ```bash
   msm worlds ram survival world
   ```
   This copies the world to `/dev/shm/msm/survival/world` and sets an `in_ram` flag.

2. **Start the server:**
   ```bash
   msm start survival
   ```
   MSM automatically creates a symlink from the world directory to the RAM copy, so Minecraft reads/writes directly to RAM.

3. **Automatic sync daemon:**
   When a server with RAM worlds starts, MSM spawns a background sync daemon (`msm-sync` screen session) that syncs RAM back to disk every 10 minutes. This runs automatically — no cron job required.

4. **On server stop:**
   When the last server stops, the sync daemon automatically terminates.

### Viewing active sessions

```bash
$ screen -ls
There are screens on:
    12345.msm-survival    (Detached)
    12346.msm-sync        (Detached)
```

### Manual sync

To force an immediate sync:
```bash
msm worlds todisk survival    # Single server
msm worlds todisk --all       # All running servers
```

### Data loss window

In the event of unexpected shutdown (power loss, kernel panic), you may lose up to 10 minutes of world changes. The sync interval is not currently configurable but can be changed by modifying `SyncIntervalSecs` in `internal/server/sync.go`.

### Differences from bash MSM

The original bash MSM required a cron job (`/etc/cron.d/msm`) for periodic syncing. The Go rewrite manages this automatically via the `msm-sync` screen session, eliminating the need for system-level cron configuration.

## Command Reference

### Global Commands

| Command | Description |
|---------|-------------|
| `msm start` | Start all servers |
| `msm stop [--now]` | Stop all servers |
| `msm restart [--now]` | Restart all servers |
| `msm config` | Display global configuration |
| `msm version` | Print version number |
| `msm cron generate` | Output cron file based on config |
| `msm cron install` | Install cron file to /etc/cron.d/msm |
| `msm logroll [--all]` | Archive server logs |

### Server Commands

| Command | Description |
|---------|-------------|
| `msm server list` | List all servers |
| `msm server create <name>` | Create a new server |
| `msm server delete <name>` | Delete a server |
| `msm server rename <old> <new>` | Rename a server |
| `msm start <server>` | Start a server |
| `msm stop <server> [--now]` | Stop a server |
| `msm restart <server> [--now]` | Restart a server |
| `msm status <server>` | Show server status |
| `msm console <server>` | Attach to server console |
| `msm cmd <server> <command>` | Send command to console |
| `msm say <server> <message>` | Broadcast message |
| `msm kick <server> <player> [reason]` | Kick a player |
| `msm config <server> [key] [value]` | Show/set server config |
| `msm backup <server>` | Full server backup |

### World Commands

| Command | Description |
|---------|-------------|
| `msm worlds list <server>` | List all worlds |
| `msm worlds on <server> <world>` | Activate a world |
| `msm worlds off <server> <world>` | Deactivate a world |
| `msm worlds ram <server> <world>` | Toggle RAM disk state |
| `msm worlds todisk [server] [--all]` | Sync RAM worlds to disk |
| `msm worlds backup [server] [--all]` | Backup all worlds |

### Player Commands

| Command | Description |
|---------|-------------|
| `msm allowlist on <server>` | Enable allowlist |
| `msm allowlist off <server>` | Disable allowlist |
| `msm allowlist add <server> <player>...` | Add to allowlist |
| `msm allowlist remove <server> <player>...` | Remove from allowlist |
| `msm allowlist list <server>` | List allowlist |
| `msm op <server> <player>` | Make player an operator |
| `msm deop <server> <player>` | Remove operator status |
| `msm blocklist player add <server> <player>... [--reason]` | Ban players |
| `msm blocklist player remove <server> <player>...` | Unban players |
| `msm blocklist player list <server>` | List banned players |
| `msm blocklist ip add <server> <ip>... [--reason]` | Ban IPs |
| `msm blocklist ip remove <server> <ip>...` | Unban IPs |
| `msm blocklist ip list <server>` | List banned IPs |

### Jar Group Commands

| Command | Description |
|---------|-------------|
| `msm jargroup list` | List all jar groups |
| `msm jargroup create <name> <url>` | Create a jar group |
| `msm jargroup delete <name>` | Delete a jar group |
| `msm jargroup rename <old> <new>` | Rename a jar group |
| `msm jargroup changeurl <name> <url>` | Change jar group URL |
| `msm jargroup getlatest <name>` | Download latest jar |
| `msm jar <server> <jargroup> [file]` | Link server to jar |

## Configuration

MSM reads its configuration from `/etc/msm.conf` (or the path specified by `--config` or the `MSM_CONF` environment variable).

### Key settings

```bash
# User to run servers as
USERNAME="minecraft"

# Storage paths
SERVER_STORAGE_PATH="/opt/msm/servers"
JAR_STORAGE_PATH="/opt/msm/jars"
WORLD_ARCHIVE_PATH="/opt/msm/archives/worlds"
BACKUP_ARCHIVE_PATH="/opt/msm/archives/backups"
LOG_ARCHIVE_PATH="/opt/msm/archives/logs"

# RAM disk (optional)
RAMDISK_STORAGE_ENABLED="true"
RAMDISK_STORAGE_PATH="/dev/shm/msm"

# Defaults
DEFAULT_RAM="1024"
DEFAULT_STOP_DELAY="10"
DEFAULT_RESTART_DELAY="10"

# Cron schedule (regenerate with: sudo msm cron install)
CRON_MSM_BINARY="/usr/local/bin/msm"
CRON_MAINTENANCE_HOUR="5"
CRON_ARCHIVE_RETENTION_DAYS="30"
```

### Per-server configuration

Each server can override defaults in `<server>/server.conf`:

```bash
USERNAME="minecraft"
RAM="2048"
JAR_PATH="paper.jar"
STOP_DELAY="30"
```

## Requirements

### Platform Support

| Platform | Status | Notes |
|----------|--------|-------|
| Linux | Full support | Primary target platform |
| macOS | Full support | Requires GNU screen (`brew install screen`) |
| Windows | WSL only | Native Windows not supported |

MSM requires a Unix-like environment due to its reliance on:
- GNU screen for process management
- POSIX filesystem semantics
- `/dev/shm` or equivalent for RAM disk features

### Minecraft Version Support

MSM supports **Minecraft 1.7.0 and later** (Java Edition). This includes all modern versions through the upcoming year-based versioning (e.g., `25.1`).

Servers running Minecraft 1.6.x or earlier are no longer supported. The original bash MSM supported 1.2.0+, but the Go rewrite drops pre-1.7 support to avoid complexity of handling differences in log file location and format.

### Dependencies

- **GNU screen** (required) - console/process management
- **rsync** (optional) - rsync backups, RAM disk sync
- **rdiff-backup** (optional) - incremental backups with rotation

## Development

```bash
# Run tests
make test

# Run linters
make lint

# Build for all platforms
make build-all
```

## Versioning

MSM uses [semantic versioning](http://semver.org/):

- **Major:** Breaking changes to configuration or command structure
- **Minor:** New features, backward compatible
- **Patch:** Bug fixes

## License

MSM is released under the [GPLv3 license](LICENSE.markdown).

## Acknowledgements

This project grew out of [Ahtenus' Minecraft Init Script](https://github.com/Ahtenus/minecraft-init). The Go rewrite maintains compatibility with the original bash MSM configuration format.

## Support

- [Submit an issue](https://github.com/msmhq/msm/issues) for bugs or feature requests
- See the [changelog](CHANGELOG.markdown) for version history
