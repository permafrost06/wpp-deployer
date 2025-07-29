# WPP Deployer

A Go application for deploying and managing multiple WordPress sites with Docker and Nginx reverse proxy.

## Features

- Deploy multiple WordPress sites with individual databases
- Automatic Nginx reverse proxy configuration
- Site management (start, stop, delete)
- Embedded templates - no external dependencies
- Clean workspace management in `~/.wpp-deployer`
- **GitHub webhook server** for CI/CD integration
- Shell completion for improved UX

## Requirements

- Go 1.21 or later
- Docker and Docker Compose
- Linux/macOS (tested on Arch Linux)
- systemd (for webhook service background operation)

## Installation

### Option 1: Using Make (Recommended)

```bash
# Build and install everything (binary + shell completions)
make install-all

# Or install components separately:
make install              # Install binary only
make install-completions  # Install shell completions only

# Set up the workspace
wpp-deployer install
```

### Option 2: Manual Installation

```bash
# Build the binary
go build -o wpp-deployer main.go

# Copy to system PATH
sudo cp wpp-deployer /usr/local/bin/

# Install shell completions (optional)
chmod +x install-completions.sh
./install-completions.sh

# Set up the workspace
wpp-deployer install
```

## Shell Completion

The application includes shell completion scripts for bash and zsh that provide:

- **Command completion**: Tab-complete `install`, `deploy`, `delete`, `list`, `exec`, `exec-all`
- **Site name completion**: Automatically complete site names for `deploy`, `delete`, and `exec` commands
- **Docker-compose command completion**: Complete common docker-compose commands like `up`, `down`, `ps`, `logs`
- **Flag completion**: Complete the `-r` flag and common docker-compose flags

### Completion Features

```bash
# Tab completion examples:
wpp-deployer <TAB>                    # Shows: install deploy delete list exec exec-all help version
wpp-deployer deploy <TAB>             # Shows: available site names
wpp-deployer exec <TAB>               # Shows: -r and available site names  
wpp-deployer exec mysite <TAB>        # Shows: up down restart ps logs etc.
wpp-deployer exec-all <TAB>           # Shows: -r up down restart ps logs etc.
```

### Manual Completion Installation

If you didn't use `make install-all`, you can install completions manually:

```bash
# Install completions
./install-completions.sh

# Or copy manually:
# Bash: copy completions/wpp-deployer.bash to your bash-completion directory
# Zsh: copy completions/_wpp-deployer to your zsh site-functions directory
```

## Usage

### Initial Setup

After installation, run the install command to set up the workspace:

```bash
wpp-deployer install
```

This will:
- Create `~/.wpp-deployer` directory
- Set up Nginx container and network
- Create necessary configuration files

### Deploy a New Site

```bash
wpp-deployer deploy mysite
```

This creates a WordPress site accessible at `mysite.nshlog.com` (assuming proper DNS configuration).

### Delete a Site

```bash
wpp-deployer delete mysite
```

Removes the site and all associated data after confirmation.

### List Sites

```bash
wpp-deployer list
```

Lists all WordPress sites (both activated and deactivated).

### Manage Sites

```bash
# Start a specific site
wpp-deployer exec mysite up -d

# Stop a specific site
wpp-deployer exec mysite down

# Stop a site and remove volumes
wpp-deployer exec mysite down --volumes

# Restart a site and reload nginx
wpp-deployer exec -r mysite restart

# Check status of a specific site
wpp-deployer exec mysite ps

# Start all sites
wpp-deployer exec-all up -d

# Stop all sites and reload nginx
wpp-deployer exec-all -r down

# Check status of all sites
wpp-deployer exec-all ps
```

## GitHub Webhook Server

The application includes a webhook server for GitHub integration that can listen for repository events like pull requests and pushes.

### Start Webhook Server

```bash
# Start webhook server (foreground)
wpp-deployer listen --port 3000 --secret your-webhook-secret

# Or run as systemd service (background)
sudo make install-service        # Install service
sudo systemctl enable wpp-deployer-webhook  # Enable service
sudo systemctl start wpp-deployer-webhook   # Start service
sudo systemctl status wpp-deployer-webhook  # Check status
sudo journalctl -u wpp-deployer-webhook -f  # View logs
```

### GitHub Configuration

1. **Configure webhook in your GitHub repository:**
   - Go to Settings → Webhooks → Add webhook
   - Payload URL: `http://your-domain.com/webhook`
   - Content type: `application/json`
   - Secret: (same as `--secret` parameter)
   - Events: Select "Pull requests" and "Pushes"

2. **Nginx automatically proxies `/webhook` to the server** (port 3000 by default)

### Webhook Events

The server listens for and displays:
- **Pull Requests**: Shows PR number, action, title, repository, and branches
- **Push Events**: Shows repository, branch, and commit range
- **Ping Events**: Confirms webhook configuration

Example output:
```
[15:30:45] 🔀 PR #123 opened: Add new feature
           Repository: username/my-repo
           Branch: feature-branch → main
           URL: https://github.com/username/my-repo/pull/123

[15:31:20] 📤 Push to username/my-repo
           Branch: main
           Commits: abc12345...def67890
```

### Service Management

```bash
# Install everything (binary + completions + service)
sudo make install-all

# Individual service operations
sudo make install-service       # Install service only

# Then manage with systemctl:
sudo systemctl enable wpp-deployer-webhook   # Enable auto-start
sudo systemctl start wpp-deployer-webhook    # Start service
sudo systemctl stop wpp-deployer-webhook     # Stop service
sudo systemctl status wpp-deployer-webhook   # Check status
sudo systemctl disable wpp-deployer-webhook  # Disable auto-start

# View logs
sudo journalctl -u wpp-deployer-webhook -f

# Complete removal
sudo make uninstall-all         # Remove binary and completions
# Manually remove service if needed:
sudo systemctl stop wpp-deployer-webhook
sudo systemctl disable wpp-deployer-webhook
sudo rm /etc/systemd/system/wpp-deployer-webhook.service
sudo systemctl daemon-reload
```

### Other Commands

```bash
# Show help
wpp-deployer help

# Show version
wpp-deployer version
```

## Architecture

The application creates the following structure in `~/.wpp-deployer`:

```
~/.wpp-deployer/
├── nginx-docker-compose.yml    # Main Nginx container
├── html/                       # Static files for main domain
│   └── index.html
├── nginx-config/               # Nginx configurations
│   ├── wpp-deployer.conf      # Main domain config
│   └── *.conf                 # Site-specific configs
├── templates/                  # Editable template files
│   ├── docker-compose.yml.template
│   ├── nginx-config.conf.template
│   ├── nginx-docker-compose.yml.template
│   ├── wpp-deployer.conf.template
│   └── index.html.template
└── wordpress-*/               # Individual WordPress sites
    ├── docker-compose.yml     # Site containers
    └── wp-data/              # WordPress files
```

## Technical Implementation

- **Pure Go**: Uses only Go standard library
- **External Templates**: Configuration templates stored in `~/.wpp-deployer/templates/` for easy editing
- **Docker Integration**: Manages Docker containers and networks
- **Template Processing**: Uses `text/template` for configuration generation
- **File Operations**: Handles workspace setup and management
- **Process Execution**: Runs Docker commands via `os/exec`

## Template Customization

Templates are stored in `~/.wpp-deployer/templates/` and can be modified after installation:

- `docker-compose.yml.template` - WordPress site container configuration
- `nginx-config.conf.template` - Nginx reverse proxy configuration per site
- `nginx-docker-compose.yml.template` - Main Nginx container configuration
- `wpp-deployer.conf.template` - Main domain Nginx configuration
- `index.html.template` - Default index page

After modifying templates, new deployments will use the updated configurations. Existing sites won't be affected unless redeployed.

## Development

### Building

```bash
make build
```

### Installing/Uninstalling

```bash
# Complete installation
make install-all

# Individual components
make install              # Binary only
make install-completions  # Shell completions only

# Uninstall
make uninstall-all        # Remove everything
make uninstall            # Remove binary only
make uninstall-completions # Remove completions only
```

### Cleaning

```bash
make clean
```

## Migration from Bash Scripts

This Go application replaces the original bash scripts:
- `deploy-site.sh` → `wpp-deployer deploy`
- `delete-site.sh` → `wpp-deployer delete`
- `site-control.sh` → `wpp-deployer exec/exec-all`

The new `exec` and `exec-all` commands provide more flexibility than the original up/down functionality, allowing any docker-compose command to be run on sites. All functionality is preserved with improved error handling and cross-platform compatibility.

## License

MIT License 