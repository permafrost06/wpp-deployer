# WPP Deployer

A Go application for deploying and managing multiple WordPress sites with Docker and Nginx reverse proxy.

## Features

- Deploy multiple WordPress sites with individual databases
- Automatic Nginx reverse proxy configuration
- Site management (start, stop, delete)
- Embedded templates - no external dependencies
- Clean workspace management in `~/.wpp-deployer`

## Requirements

- Go 1.21 or later
- Docker and Docker Compose
- Linux/macOS (tested on Arch Linux)

## Installation

### Option 1: Using Make (Recommended)

```bash
# Build and install the binary
make install

# Set up the workspace
wpp-deployer install
```

### Option 2: Manual Installation

```bash
# Build the binary
go build -o wpp-deployer main.go

# Copy to system PATH
sudo cp wpp-deployer /usr/local/bin/

# Set up the workspace
wpp-deployer install
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

### Cleaning

```bash
make clean
```

### Uninstalling

```bash
make uninstall
```

## Migration from Bash Scripts

This Go application replaces the original bash scripts:
- `deploy-site.sh` → `wpp-deployer deploy`
- `delete-site.sh` → `wpp-deployer delete`
- `site-control.sh` → `wpp-deployer exec/exec-all`

The new `exec` and `exec-all` commands provide more flexibility than the original up/down functionality, allowing any docker-compose command to be run on sites. All functionality is preserved with improved error handling and cross-platform compatibility.

## License

MIT License 