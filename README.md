# WordPress Plugin Deployer GitHub App

This app listens for GitHub push and pull request events on WordPress plugin repositories. On each event, it:

1. Clones the repository at the relevant commit/branch/PR.
2. Runs the plugin's npm/pnpm build command to create a plugin zip.
3. Spins up a fresh WordPress site in Docker (one per branch/PR).
4. Installs and activates the plugin zip on that site using WP-CLI.

## Prerequisites
- Go 1.21+
- Docker & Docker Compose
- npm and/or pnpm (globally installed)
- GitHub App (with webhook secret, repo read access)

## Setup
1. **Clone this repo**
2. **Install Go dependencies:**
   ```sh
   go mod tidy
   ```
3. **Configure your GitHub App:**
   - Set webhook URL to `http://your-server/webhook`
   - Subscribe to `push` and `pull_request` events
   - Give read access to code

4. **Environment variables:**
   - `PORT` (optional): Port to run the server (default: 8080)

## Usage
Run the server:
```sh
go run .
```

## How it works
- On push/PR, the app clones the repo, runs `npm run build:zip` (or `pnpm run build:zip`), finds the resulting zip, and deploys a new WordPress site using Docker Compose.
- The plugin zip is copied into the container and installed/activated with WP-CLI.
- Each branch/PR gets its own site (named by repo and branch/pr).

## Notes
- The plugin repo must provide a `build:zip` script in its `package.json`.
- Docker Compose files are generated per site from a template.
- Sites are exposed on random high ports (20k-30k).

## TODO
- Add authentication for webhooks (verify GitHub signature)
- Clean up old containers/sites
- Add status reporting 