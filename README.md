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

## Repository Configuration

The application can store build configurations for repositories to enable automated deployments via webhooks.

### Add Repository Configuration

```bash
# Add a new repository configuration
wpp-deployer add-repo username/repo-name "build-command" "zip-location"

# Examples:
wpp-deployer add-repo myuser/frontend "pnpm i && pnpm run build" "dist/app.zip"
wpp-deployer add-repo frost/tableberg "pnpm i && pnpm run export" "packages/tableberg/tableberg.zip"
wpp-deployer add-repo company/api "npm ci && npm run compile" "build/api.zip"
```

### List Repository Configurations

```bash
wpp-deployer list-repos
```

Example output:
```
Configured repositories (2):

  frost/tableberg
    Build: pnpm i && pnpm run export
    Zip:   packages/tableberg/tableberg.zip

  myuser/frontend
    Build: pnpm i && pnpm run build
    Zip:   dist/app.zip
```

### Repository Configuration Storage

- Configurations are stored in `~/.wpp-deployer/repos.json`
- Repository format must be `username/repo-name`
- Build commands can include multiple steps with `&&`
- Zip location should be relative to the repository root

> **Note**: Repository configurations are prepared for future webhook-triggered automated deployments. When a push or pull request event is received for a configured repository, the system will be able to automatically run the build command and deploy the resulting zip file.

### Prerequisites for Automated Deployment

The webhook server requires the following tools to be installed:
- `git` - For cloning and updating repositories
- `bash` - For executing build commands
- WordPress sites must have WP-CLI available (included in the Docker setup)

### File Structure for Automated Deployments

When webhooks trigger automated deployments, the following structure is created in `~/.wpp-deployer/`:

```
~/.wpp-deployer/
├── repos/                      # Cloned repositories for builds
│   └── username/
│       └── repo-name/          # Repository contents
├── wordpress-username-repo-name-branch/  # Generated WordPress sites
│   ├── docker-compose.yml
│   └── wp-data/
│       └── wp-content/
│           └── plugins/        # Installed plugins appear here
└── repos.json                 # Repository configurations
```

Example site URLs generated:
- `frost-tableberg-main.nshlog.com` (from push to main branch)
- `frost-tableberg-feature-branch.nshlog.com` (from PR with feature-branch)
- `myuser-frontend-dev.nshlog.com` (from push to dev branch)

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

**Automatic Deployment**: When push or PR events are received for repositories configured with `add-repo`, the system automatically:

1. **Matches Repository**: Checks if the webhook repository matches a configured repo
2. **Creates Site**: Generates site name as `username-repo-name-branch.nshlog.com`
3. **Clones Repository**: Downloads/updates code to `~/.wpp-deployer/repos/username/repo-name`
4. **Runs Build**: Executes the configured build command
5. **Deploys WordPress**: Creates/updates WordPress site
6. **Installs Plugin**: Uses WP-CLI to install and activate the built plugin zip file

> **Note**: Repositories not configured with `add-repo` will only display webhook events in the console without triggering deployments.

Example output:
```
[15:30:45] 📤 Push to frost/tableberg
         Branch: main
         Commits: abc12345...def67890
         [+] Repository configured for deployment
         [+] Build: pnpm i && pnpm run export
         [+] Zip: packages/tableberg/tableberg.zip
         [+] Creating site: frost-tableberg-main.nshlog.com
         [+] Updating repository...
         [+] Cloning repository...
         [+] Setting up WordPress site...
         [+] Creating new WordPress site: frost-tableberg-main
         [+] Running build command: pnpm i && pnpm run export
         [+] Installing WordPress plugin...
         [+] Installing plugin: packages/tableberg/tableberg.zip
         [+] Plugin installed and activated successfully!
         [✔] Deployment completed successfully!

[15:31:20] 🔀 PR #123 opened: Add new feature
         Repository: frost/tableberg
         Branch: feature-branch → main
         URL: https://github.com/frost/tableberg/pull/123
         [+] Repository configured for deployment
         [+] Build: pnpm i && pnpm run export
         [+] Zip: packages/tableberg/tableberg.zip
         [+] Creating site: frost-tableberg-feature-branch.nshlog.com
         [+] Installing WordPress plugin...
         [+] Plugin installed and activated successfully!
         [✔] Deployment completed successfully!
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
- `