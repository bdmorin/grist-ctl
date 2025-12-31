# Gristle - The Meaty CLI for Grist

```
   _____ _____  _____  _____ _______ _      ______
  / ____|  __ \|_   _|/ ____|__   __| |    |  ____|
 | |  __| |__) | | | | (___    | |  | |    | |__
 | | |_ |  _  /  | |  \___ \   | |  | |    |  __|
 | |__| | | \ \ _| |_ ____) |  | |  | |____| |____
  \_____|_|  \_\_____|_____/   |_|  |______|______|

      The tough, chewy bits that get the job done.
```

> *Fork of the excellent [grist-ctl](https://github.com/Ville-Eurometropole-Strasbourg/grist-ctl) by Ville Eurometropole Strasbourg*

**[Grist](https://www.getgrist.com/)** is a versatile platform for creating and managing custom data applications. It blends the capabilities of a relational database with the adaptability of a spreadsheet, empowering users to design advanced data workflows, collaborate in real-time, and automate tasks--all without requiring code.

**Gristle** is the tough, no-nonsense command-line utility for wrangling your Grist data. Like the chewy bits of a good steak, it's not glamorous, but it gets the job done. Automate document management, export data, manage users, and more--all from your terminal.

<div align="center">

[Installation](#installation) |
[Configuration](#configuration) |
[Usage](#usage) |
[Interactive TUI](#interactive-tui) |
[MCP Server](#mcp-server)

</div>

## Features

- **Interactive TUI** - Navigate your Grist orgs, workspaces, and docs with a beautiful terminal interface
- **MCP Server** - AI assistant integration via Model Context Protocol
- **CLI Commands** - Script everything for automation
- **Multiple Output Formats** - Table, JSON, or CSV output

## Installation

### Pre-built Binaries

Grab the latest release for your platform:

```bash
# Linux
curl -L https://github.com/bdmorin/gristle/releases/latest/download/gristle-linux-amd64 -o gristle
chmod +x gristle
sudo mv gristle /usr/local/bin/

# macOS (Apple Silicon)
curl -L https://github.com/bdmorin/gristle/releases/latest/download/gristle-darwin-arm64 -o gristle
chmod +x gristle
sudo mv gristle /usr/local/bin/

# Windows - download gristle.exe and add to PATH
```

### Building from Source

```bash
# Clone it
git clone https://github.com/bdmorin/gristle.git
cd gristle

# Build it
go build -o gristle .

# Run it
./gristle
```

**Requirements:** Go 1.24+

## Configuration

Gristle needs your Grist server URL and API token. Get your token from your Grist profile settings.

### Interactive Setup

```bash
$ gristle config
---------------------------------------------------------------------------
Setting the url and token for access to the grist server (/Users/you/.gristle)
---------------------------------------------------------------------------
Actual URL : (none)
Token : (none)
Would you like to configure (Y/N) ? y
Grist server URL: https://docs.getgrist.com
User token: your-secret-token-here
Config saved in /Users/you/.gristle
```

### Manual Setup

Create `~/.gristle`:

```ini
GRIST_URL="https://docs.getgrist.com"
GRIST_TOKEN="your-secret-token-here"
```

## Usage

### Interactive TUI

Just run `gristle` with no arguments for the interactive terminal UI:

```bash
$ gristle
```

Navigate with arrow keys, Enter to select, Esc to go back, q to quit.

### MCP Server

Start the MCP server for AI assistant integration:

```bash
$ gristle mcp
# or
$ gristle serve
```

### CLI Commands

```bash
gristle [flags] <command> [subcommand] [args]
```

#### Global Flags

| Flag | Description |
|------|-------------|
| `-o, --output` | Output format: `table` (default) or `json` |
| `--json` | Shorthand for `-o json` |
| `-h, --help` | Help for any command |

#### Commands

**General**
| Command | Description |
|---------|-------------|
| `gristle config` | Configure Grist server URL & token |
| `gristle version` | Show version information |
| `gristle help [command]` | Get help for any command |

**Organizations**
| Command | Description |
|---------|-------------|
| `gristle org list` | List all organizations |
| `gristle org get <id>` | Get organization details |
| `gristle org access <id>` | Show organization member access |
| `gristle org usage <id>` | Show organization usage stats |
| `gristle create org <name> <domain>` | Create a new organization |
| `gristle delete org <id> <name>` | Delete an organization |

**Workspaces**
| Command | Description |
|---------|-------------|
| `gristle workspace get <id>` | Get workspace details |
| `gristle workspace access <id>` | Show workspace access permissions |
| `gristle delete workspace <id>` | Delete a workspace |

**Documents**
| Command | Description |
|---------|-------------|
| `gristle doc get <id>` | Get document details |
| `gristle doc access <id>` | Show document access permissions |
| `gristle doc webhooks <id>` | List document webhooks |
| `gristle doc table <id> <table>` | Export table as CSV |
| `gristle doc export <id> excel` | Export document as Excel |
| `gristle doc export <id> grist` | Export document as Grist (sqlite) |
| `gristle move doc <id> <wsid>` | Move document to workspace |
| `gristle move docs <from-wsid> <to-wsid>` | Move all docs between workspaces |
| `gristle purge doc <id> [keep]` | Purge doc history (default: keep 3 states) |
| `gristle delete doc <id>` | Delete a document |

**Users**
| Command | Description |
|---------|-------------|
| `gristle users list` | List all users and their roles |
| `gristle import users` | Import users from stdin |
| `gristle delete user <id>` | Delete a user |

### Examples

```bash
# List all orgs
$ gristle org list
+----+----------+
| ID |   NAME   |
+----+----------+
|  2 | Personal |
|  3 | Work     |
+----+----------+

# Get org details as JSON
$ gristle org get 3 --json

# Export a document to Excel
$ gristle doc export abc123 excel

# Move all docs from one workspace to another
$ gristle move docs 100 200

# Get help for any command
$ gristle help doc
$ gristle doc --help
```

## Contributing

We welcome contributions! See [CONTRIBUTING.md](CONTRIBUTING.md) for details.

## Credits

Gristle is a fork of [grist-ctl](https://github.com/Ville-Eurometropole-Strasbourg/grist-ctl) by [Ville Eurométropole Strasbourg](https://github.com/Ville-Eurometropole-Strasbourg). Much respect to the original authors for their excellent work.

**🇫🇷 Merci beaucoup! 🇫🇷**

## License

MIT License - see [LICENSE](LICENSE) for details.

---

*Remember: Like a good steak, your data deserves proper handling. Gristle's got you covered.*
