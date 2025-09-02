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

**For automated deployments via webhooks:**
- `git` - Repository cloning and updates
- `bash` - Build command execution
- WordPress sites with WP-CLI (included in Docker setup)

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

### Other Commands

```bash
# Show help
wpp-deployer help

# Show version
wpp-deployer version
```

## Troubleshooting

### WordPress Container Readiness Timeout

If you see: `WordPress container did not become ready within 60 seconds`

This was caused by **extremely long Docker service names** (70+ characters) preventing proper DNS resolution within Docker networks.

**Fixed in this version:**
- **Shortened service names**: Uses first 10 characters of repo name for service identifiers  
- **Maintained unique naming**: Each site still has unique services to prevent conflicts
- **Preserved descriptive containers**: Container names remain descriptive for `docker ps`

**Example transformation:**
```yaml
# OLD: Very long service names causing DNS timeouts
services:
  wordpress-permafrost06-simple-file-submission-plugin-wpp-test:
    environment:
      WORDPRESS_DB_HOST: wordpress-permafrost06-simple-file-submission-plugin-wpp-test-db

# NEW: Short service names, descriptive containers
services:
  wordpress-simple-fil:                    # First 10 chars of repo name
    container_name: wordpress-permafrost06-simple-file-submission-plugin-wpp-test-apache
    environment:
      WORDPRESS_DB_HOST: wordpress-simple-fil-db
```

### WP-CLI TTY Error in Webhook Deployments

If you see: `the input device is not a TTY` during plugin installation or WP-CLI commands

This happens when **webhook server runs WP-CLI commands** without an interactive terminal.

**Fixed in this version:**
- **Added `-T` flag**: All `docker compose run` commands now use `-T` to disable TTY allocation
- **Background compatibility**: WP-CLI commands work properly in webhook server context
- **No manual intervention**: Fully automated deployments without TTY requirements

**Commands fixed:**
```bash
# Plugin installation
docker compose run -T --rm wpcli plugin install plugin.zip --activate

# WordPress core installation  
docker compose run -T --rm wpcli --allow-root core install [options]

# Database health checks
docker compose run -T --rm wpcli --allow-root db check
```

### Nginx Server Names Hash Bucket Size Error

If you encounter the error: `nginx: [emerg] could not build server_names_hash, you should increase server_names_hash_bucket_size: 64`

This happens when you have many WordPress sites deployed. The solution is included automatically:

- **Custom nginx.conf**: Includes `server_names_hash_bucket_size 128` setting
- **Automatic Configuration**: New installations include the fix by default
- **Existing Installations**: Run `wpp-deployer install` again to update configuration

To manually fix existing installations:
```bash
# The install command will update your nginx configuration
wpp-deployer install

# Or manually restart nginx with the updated configuration
cd ~/.wpp-deployer
docker compose -f nginx-docker-compose.yml restart
```

## Architecture

The application creates the following structure in `~/.wpp-deployer`:

```
~/.wpp-deployer/
├── nginx-docker-compose.yml    # Main Nginx container
├── nginx.conf                  # Custom nginx config with optimizations
├── html/                       # Static files for main domain
│   └── index.html
├── nginx-config/               # Nginx configurations
│   ├── wpp-deployer.conf      # Main domain config
│   └── *.conf                 # Site-specific configs
├── templates/                  # Editable template files
│   ├── docker-compose.yml.template
│   ├── nginx-config.conf.template
│   ├── nginx-docker-compose.yml.template
│   ├── nginx.conf.template
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
- `nginx.conf.template` - Main nginx configuration with performance optimizations
- `wpp-deployer.conf.template` - Main domain and webhook proxy configuration
- `index.html.template` - Default static HTML page
