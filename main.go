package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strings"
	"text/template"
	"time"
)

const (
	appName = "wpp-deployer"
	version = "1.0.0"
)

// Template file names
const (
	dockerComposeTemplateFile = "docker-compose.yml.template"
	nginxConfigTemplateFile   = "nginx-config.conf.template"
	nginxDockerComposeFile    = "nginx-docker-compose.yml.template"
	mainNginxConfigFile       = "wpp-deployer.conf.template"
	nginxMainConfigFile       = "nginx.conf.template"
	indexHTMLFile             = "index.html.template"
)

type TemplateData struct {
	Sitename string
}

// generateShortName creates a shortened version of the sitename for service names
// Takes the first 10 characters and removes any non-alphanumeric characters except hyphens
func generateShortName(sitename string) string {
	// For webhook deployments, extract repo name from username-repo-branch format
	parts := strings.Split(sitename, "-")
	var repoName string
	if len(parts) >= 3 {
		// Skip username (first part) and take repo name parts
		repoName = strings.Join(parts[1:len(parts)-1], "-")
	} else {
		repoName = sitename
	}

	// Take first 10 characters
	if len(repoName) > 10 {
		repoName = repoName[:10]
	}

	// Clean up any trailing hyphens
	repoName = strings.TrimRight(repoName, "-")

	return repoName
}

type RepoConfig struct {
	Repo   string `json:"repo"`
	Script string `json:"script"`
}

type WPPDeployer struct {
	workDir string
}

func NewWPPDeployer() (*WPPDeployer, error) {
	usr, err := user.Current()
	if err != nil {
		return nil, fmt.Errorf("failed to get current user: %w", err)
	}

	workDir := filepath.Join(usr.HomeDir, ".wpp-deployer")
	return &WPPDeployer{workDir: workDir}, nil
}

func (w *WPPDeployer) loadTemplate(templateFile string) (string, error) {
	templatePath := filepath.Join(w.workDir, "templates", templateFile)
	content, err := os.ReadFile(templatePath)
	if err != nil {
		return "", fmt.Errorf("failed to read template %s: %w", templateFile, err)
	}
	return string(content), nil
}

func (w *WPPDeployer) getRepoConfigPath() string {
	return filepath.Join(w.workDir, "repos.json")
}

func (w *WPPDeployer) loadRepoConfigs() (map[string]RepoConfig, error) {
	configPath := w.getRepoConfigPath()
	configs := make(map[string]RepoConfig)

	// If file doesn't exist, return empty map
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return configs, nil
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read repo configs: %w", err)
	}

	if err := json.Unmarshal(data, &configs); err != nil {
		return nil, fmt.Errorf("failed to parse repo configs: %w", err)
	}

	return configs, nil
}

func (w *WPPDeployer) saveRepoConfigs(configs map[string]RepoConfig) error {
	configPath := w.getRepoConfigPath()

	data, err := json.MarshalIndent(configs, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal repo configs: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write repo configs: %w", err)
	}

	return nil
}

func (w *WPPDeployer) AddRepo(repo, script string) error {
	if repo == "" {
		return fmt.Errorf("repository name is required")
	}
	if script == "" {
		return fmt.Errorf("script is required")
	}

	// Validate repo format (should be username/repo-name)
	if !strings.Contains(repo, "/") {
		return fmt.Errorf("repository should be in format 'username/repo-name'")
	}

	configs, err := w.loadRepoConfigs()
	if err != nil {
		return err
	}

	// Check if repo already exists
	if _, exists := configs[repo]; exists {
		return fmt.Errorf("repository '%s' already exists", repo)
	}

	configs[repo] = RepoConfig{
		Repo:   repo,
		Script: script,
	}

	if err := w.saveRepoConfigs(configs); err != nil {
		return err
	}

	fmt.Printf("[✔] Repository configuration added:\n")
	fmt.Printf("    Repository: %s\n", repo)
	fmt.Printf("    Script: %s\n", script)

	return nil
}

func (w *WPPDeployer) ListRepos() error {
	configs, err := w.loadRepoConfigs()
	if err != nil {
		return err
	}

	if len(configs) == 0 {
		fmt.Println("No repositories configured.")
		return nil
	}

	fmt.Printf("Configured repositories (%d):\n\n", len(configs))
	for repo, config := range configs {
		fmt.Printf("  %s\n", repo)
		fmt.Printf("    Script: %s\n", config.Script)
		fmt.Println()
	}

	return nil
}

func (w *WPPDeployer) Install() error {
	fmt.Printf("[+] Installing %s to %s...\n", appName, w.workDir)

	if err := os.MkdirAll(w.workDir, 0755); err != nil {
		return fmt.Errorf("failed to create work directory: %w", err)
	}

	dirs := []string{"html", "nginx-config", "templates"}
	for _, dir := range dirs {
		dirPath := filepath.Join(w.workDir, dir)
		if err := os.MkdirAll(dirPath, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	templateFiles := map[string]string{
		"docker-compose.yml.template":       dockerComposeTemplateFile,
		"nginx-config.conf.template":        nginxConfigTemplateFile,
		"nginx-docker-compose.yml.template": nginxDockerComposeFile,
		"wpp-deployer.conf.template":        mainNginxConfigFile,
		"nginx.conf.template":               nginxMainConfigFile,
		"index.html.template":               indexHTMLFile,
	}

	for srcFile, templateFile := range templateFiles {
		srcPath := filepath.Join("templates", srcFile)
		destPath := filepath.Join(w.workDir, "templates", templateFile)

		content, err := os.ReadFile(srcPath)
		if err != nil {
			return fmt.Errorf("failed to read source template %s: %w", srcFile, err)
		}

		if err := os.WriteFile(destPath, content, 0644); err != nil {
			return fmt.Errorf("failed to write template %s: %w", templateFile, err)
		}
	}

	nginxDockerComposeContent, err := w.loadTemplate(nginxDockerComposeFile)
	if err != nil {
		return fmt.Errorf("failed to load nginx docker-compose template: %w", err)
	}

	indexHTMLContent, err := w.loadTemplate(indexHTMLFile)
	if err != nil {
		return fmt.Errorf("failed to load index.html template: %w", err)
	}

	mainNginxConfigContent, err := w.loadTemplate(mainNginxConfigFile)
	if err != nil {
		return fmt.Errorf("failed to load main nginx config template: %w", err)
	}

	nginxMainConfigContent, err := w.loadTemplate(nginxMainConfigFile)
	if err != nil {
		return fmt.Errorf("failed to load nginx main config template: %w", err)
	}

	files := map[string]string{
		"nginx-docker-compose.yml":       nginxDockerComposeContent,
		"html/index.html":                indexHTMLContent,
		"nginx-config/wpp-deployer.conf": mainNginxConfigContent,
		"nginx.conf":                     nginxMainConfigContent,
	}

	for filePath, content := range files {
		fullPath := filepath.Join(w.workDir, filePath)
		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			return fmt.Errorf("failed to create file %s: %w", filePath, err)
		}
	}

	fmt.Printf("[✔] Installation completed successfully!\n")
	fmt.Printf("    Work directory: %s\n", w.workDir)

	fmt.Println("[+] Creating Docker network...")
	cmd := exec.Command("docker", "network", "create", "wpp-deployer-network")
	if err := cmd.Run(); err != nil {
		fmt.Println("[!] Network might already exist (this is okay)")
	}

	fmt.Println("[+] Starting nginx container...")
	cmd = exec.Command("docker", "compose", "-f", filepath.Join(w.workDir, "nginx-docker-compose.yml"), "up", "-d")
	cmd.Dir = w.workDir
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to start nginx container: %w", err)
	}

	return nil
}

func (w *WPPDeployer) Deploy(sitename string) error {
	domain := fmt.Sprintf("%s.nshlog.com", sitename)
	targetDir := filepath.Join(w.workDir, fmt.Sprintf("wordpress-%s", sitename))
	nginxConfig := filepath.Join(w.workDir, "nginx-config", fmt.Sprintf("%s.conf", sitename))

	if _, err := os.Stat(targetDir); err == nil {
		return fmt.Errorf("site '%s' already exists", sitename)
	}

	fmt.Printf("[+] Creating site directory for '%s'...\n", sitename)
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return fmt.Errorf("failed to create site directory: %w", err)
	}

	wpDataDir := filepath.Join(targetDir, "wp-data")
	if err := os.MkdirAll(wpDataDir, 0755); err != nil {
		return fmt.Errorf("failed to create wp-data directory: %w", err)
	}

	fmt.Println("[+] Generating configuration files...")

	tmplData := TemplateData{
		Sitename: sitename,
	}

	dockerComposePath := filepath.Join(targetDir, "docker-compose.yml")
	if err := w.createFileFromTemplate(dockerComposeTemplateFile, dockerComposePath, tmplData); err != nil {
		return fmt.Errorf("failed to create docker-compose.yml: %w", err)
	}

	if err := w.createFileFromTemplate(nginxConfigTemplateFile, nginxConfig, tmplData); err != nil {
		return fmt.Errorf("failed to create nginx config: %w", err)
	}

	fmt.Printf("[+] Starting containers for '%s'...\n", sitename)
	cmd := exec.Command("docker", "compose", "-f", dockerComposePath, "up", "-d")
	cmd.Dir = targetDir
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to start containers: %w", err)
	}

	fmt.Println("[+] Installing WordPress...")
	if err := w.installWordPress(sitename, targetDir); err != nil {
		return fmt.Errorf("failed to install WordPress: %w", err)
	}

	fmt.Println("[+] Reloading nginx...")
	if err := w.reloadNginx(); err != nil {
		return fmt.Errorf("failed to reload nginx: %w", err)
	}

	fmt.Printf("[✔] Site '%s' deployed successfully!\n", domain)
	return nil
}

func (w *WPPDeployer) Delete(sitename string) error {
	targetDir := filepath.Join(w.workDir, fmt.Sprintf("wordpress-%s", sitename))
	nginxConfig := filepath.Join(w.workDir, "nginx-config", fmt.Sprintf("%s.conf", sitename))

	if _, err := os.Stat(targetDir); os.IsNotExist(err) {
		return fmt.Errorf("site directory '%s' does not exist", sitename)
	}

	fmt.Printf("Are you sure you want to delete the site '%s'? This will remove all data. (y/N): ", sitename)
	reader := bufio.NewReader(os.Stdin)
	response, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("failed to read input: %w", err)
	}

	response = strings.TrimSpace(strings.ToLower(response))
	if response != "y" && response != "yes" {
		fmt.Println("Aborted.")
		return nil
	}

	fmt.Println("[−] Stopping and removing containers...")
	dockerComposePath := filepath.Join(targetDir, "docker-compose.yml")
	cmd := exec.Command("docker", "compose", "-f", dockerComposePath, "down", "--volumes")
	cmd.Dir = targetDir
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to stop containers: %w", err)
	}

	fmt.Println("[−] Removing nginx config...")
	if err := os.Remove(nginxConfig); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove nginx config: %w", err)
	}

	fmt.Println("[↻] Reloading nginx...")
	if err := w.reloadNginx(); err != nil {
		return fmt.Errorf("failed to reload nginx: %w", err)
	}

	fmt.Println("[−] Deleting site directory...")
	if err := os.RemoveAll(targetDir); err != nil {
		fmt.Println("[!] Normal deletion failed (likely due to Docker container file ownership)")
		fmt.Println("[+] Attempting deletion with elevated privileges...")

		cmd := exec.Command("sudo", "rm", "-rf", targetDir)
		if sudoErr := cmd.Run(); sudoErr != nil {
			return fmt.Errorf("failed to delete site directory with sudo: %w (original error: %v)", sudoErr, err)
		}
		fmt.Println("[✔] Directory deleted with elevated privileges")
	}

	fmt.Printf("[✔] Site '%s' deleted successfully.\n", sitename)
	return nil
}

func (w *WPPDeployer) Exec(sitename string, args []string, reloadNginx bool) error {
	targetDir := filepath.Join(w.workDir, fmt.Sprintf("wordpress-%s", sitename))
	dockerComposePath := filepath.Join(targetDir, "docker-compose.yml")

	if _, err := os.Stat(targetDir); os.IsNotExist(err) {
		return fmt.Errorf("site directory 'wordpress-%s' not found", sitename)
	}

	if _, err := os.Stat(dockerComposePath); os.IsNotExist(err) {
		return fmt.Errorf("docker-compose.yml not found for site '%s'", sitename)
	}

	fmt.Printf("[•] Running docker compose command on wordpress-%s...\n", sitename)

	cmdArgs := []string{"compose", "-f", dockerComposePath}
	cmdArgs = append(cmdArgs, args...)

	cmd := exec.Command("docker", cmdArgs...)
	cmd.Dir = targetDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("docker compose command failed for site %s: %w", sitename, err)
	}

	if reloadNginx {
		fmt.Println("[↻] Reloading nginx...")
		if err := w.reloadNginx(); err != nil {
			return fmt.Errorf("failed to reload nginx: %w", err)
		}
	}

	return nil
}

func (w *WPPDeployer) ExecAll(args []string, reloadNginx bool) error {
	sites, err := w.List()
	if err != nil {
		return fmt.Errorf("failed to get sites list: %w", err)
	}

	if len(sites) == 0 {
		fmt.Println("[!] No sites found")
		return nil
	}

	fmt.Printf("[*] Running docker compose command on %d sites...\n", len(sites))

	for _, sitename := range sites {
		if err := w.Exec(sitename, args, false); err != nil {
			fmt.Printf("[!] Error with site %s: %v\n", sitename, err)
		}
	}

	if reloadNginx {
		fmt.Println("[↻] Reloading nginx...")
		if err := w.reloadNginx(); err != nil {
			return fmt.Errorf("failed to reload nginx: %w", err)
		}
	}

	return nil
}

func (w *WPPDeployer) List() ([]string, error) {
	var sites []string

	err := filepath.WalkDir(w.workDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() && strings.HasPrefix(d.Name(), "wordpress-") {
			dockerComposePath := filepath.Join(path, "docker-compose.yml")
			if _, err := os.Stat(dockerComposePath); err == nil {
				siteName := strings.TrimPrefix(d.Name(), "wordpress-")
				sites = append(sites, siteName)
			}
		}
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to scan for sites: %w", err)
	}

	return sites, nil
}

func (w *WPPDeployer) createFileFromTemplate(templateFile, outputPath string, data TemplateData) error {
	tmplContent, err := w.loadTemplate(templateFile)
	if err != nil {
		return fmt.Errorf("failed to load template: %w", err)
	}

	tmpl, err := template.New("config").Parse(tmplContent)
	if err != nil {
		return fmt.Errorf("failed to parse template: %w", err)
	}

	file, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	if err := tmpl.Execute(file, data); err != nil {
		return fmt.Errorf("failed to execute template: %w", err)
	}

	return nil
}

func (w *WPPDeployer) reloadNginx() error {
	// Try config test with retries to allow upstream containers to be ready
	for i := 0; i < 5; i++ {
		cmd := exec.Command("docker", "exec", "wpp-deployer-nginx", "nginx", "-t")
		if err := cmd.Run(); err == nil {
			// Config test passed, reload nginx
			cmd = exec.Command("docker", "exec", "wpp-deployer-nginx", "nginx", "-s", "reload")
			return cmd.Run()
		}

		if i < 4 { // Don't sleep on the last attempt
			fmt.Printf("[•] Nginx config test failed, retrying in 2 seconds... (%d/5)\n", i+1)
			time.Sleep(2 * time.Second)
		}
	}

	fmt.Println("[!] Skipped nginx reload: config test failed after 5 attempts")
	return nil
}

func (w *WPPDeployer) installWordPress(sitename, targetDir string) error {
	dockerComposePath := filepath.Join(targetDir, "docker-compose.yml")
	wordpressService := fmt.Sprintf("wordpress-%s", sitename)
	dbService := fmt.Sprintf("wordpress-%s-db", sitename)

	// Wait for WordPress container to be ready
	fmt.Println("[•] Waiting for WordPress container to be ready...")
	for i := 0; i < 60; i++ { // Wait up to 60 seconds
		// Check if both WordPress and DB services are running
		cmd := exec.Command("docker", "compose", "-f", dockerComposePath, "ps", "--services", "--filter", "status=running")
		cmd.Dir = targetDir
		output, err := cmd.Output()
		if err == nil {
			outputStr := string(output)
			// Check if both services are running
			if strings.Contains(outputStr, wordpressService) && strings.Contains(outputStr, dbService) {
				// Also check if MySQL is ready by testing WP-CLI connection
				cmd = exec.Command("docker", "compose", "-f", dockerComposePath, "run", "-T", "--rm", "wpcli", "--allow-root", "db", "check")
				cmd.Dir = targetDir
				if err := cmd.Run(); err == nil {
					break // Both WordPress and database are ready
				}
			}
		}

		if i == 59 {
			return fmt.Errorf("WordPress container did not become ready within 60 seconds")
		}

		fmt.Printf(".")
		time.Sleep(1 * time.Second)
	}
	fmt.Println()

	// Check if WordPress is already installed
	fmt.Println("[•] Checking if WordPress is already installed...")
	cmd := exec.Command("docker", "compose", "-f", dockerComposePath, "run", "-T", "--rm", "wpcli", "--allow-root", "core", "is-installed")
	cmd.Dir = targetDir
	if err := cmd.Run(); err == nil {
		fmt.Println("[!] WordPress is already installed, skipping installation")
		return nil
	}

	// Install WordPress using the dedicated wpcli service
	fmt.Println("[•] Installing WordPress core...")
	url := fmt.Sprintf("http://%s.nshlog.com", sitename)
	cmd = exec.Command("docker", "compose", "-f", dockerComposePath, "run", "-T", "--rm", "wpcli",
		"--allow-root", "core", "install",
		fmt.Sprintf("--url=%s", url),
		fmt.Sprintf("--title=%s", sitename),
		"--admin_user=outmatch-underdog",
		"--admin_password=7E3cdGT0EyucyA",
		"--admin_email=admin@nshlog.com")
	cmd.Dir = targetDir

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to run WordPress installation: %w", err)
	}

	fmt.Println("[✔] WordPress installed successfully")
	return nil
}

func printUsage() {
	fmt.Printf(`%s v%s - WordPress Site Deployer

Usage:
  %s <command> [options] [arguments]

Commands:
  install                           Set up %s workspace in ~/.%s
  deploy <sitename>                 Deploy a new WordPress site
  delete <sitename>                 Delete an existing WordPress site
  list                              List all WordPress sites
  exec [-r] <sitename> <args...>    Run docker-compose command on specific site
  exec-all [-r] <args...>           Run docker-compose command on all sites
  listen [--port PORT] [--secret SECRET]  Start webhook server for GitHub events
  add-repo <username/repo> <script>  Add a new repository for deployment
  list-repos                          List all configured repositories

Options:
  -r                   Reload nginx after command execution
  --port PORT          Webhook server port (default: 3000)
  --secret SECRET      GitHub webhook secret for validation

Examples:
  %s install
  %s deploy mysite
  %s delete mysite
  %s list
  %s exec mysite up -d
  %s exec -r mysite down --volumes
  %s exec mysite ps
  %s exec-all -r restart
  %s exec-all ps
  %s listen --port 3000 --secret mysecret
  %s add-repo myuser/myapp 'npm run build && wp plugin install \\$REPO_PATH/dist/plugin.zip --activate'
  %s add-repo myuser/site 'npm run build && wp theme install \\$REPO_PATH/dist/theme.zip --activate'
  %s list-repos

`, appName, version, appName, appName, appName, appName, appName, appName, appName, appName, appName, appName, appName, appName, appName, appName, appName, appName)
}

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	command := os.Args[1]

	deployer, err := NewWPPDeployer()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	switch command {
	case "install":
		if err := deployer.Install(); err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}

	case "deploy":
		if len(os.Args) < 3 {
			fmt.Println("Error: sitename is required for deploy command")
			printUsage()
			os.Exit(1)
		}

		sitename := os.Args[2]
		if sitename == "" {
			fmt.Println("Error: sitename is required for deploy command")
			os.Exit(1)
		}

		if err := deployer.Deploy(sitename); err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}

	case "delete":
		if len(os.Args) < 3 {
			fmt.Println("Error: sitename is required for delete command")
			printUsage()
			os.Exit(1)
		}
		sitename := os.Args[2]
		if sitename == "" {
			fmt.Println("Error: sitename is required for deploy command")
			os.Exit(1)
		}

		if err := deployer.Delete(sitename); err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}

	case "exec":
		args := os.Args[2:]
		reloadNginx := false

		if len(args) > 0 && args[0] == "-r" {
			reloadNginx = true
			args = args[1:]
		}

		if len(args) < 2 {
			fmt.Println("Error: exec requires sitename and docker-compose arguments")
			fmt.Println("Usage: wpp-deployer exec [-r] <sitename> <docker-compose-args...>")
			os.Exit(1)
		}

		sitename := args[0]
		composeArgs := args[1:]

		if err := deployer.Exec(sitename, composeArgs, reloadNginx); err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}

	case "exec-all":
		args := os.Args[2:]
		reloadNginx := false

		if len(args) > 0 && args[0] == "-r" {
			reloadNginx = true
			args = args[1:]
		}

		if len(args) < 1 {
			fmt.Println("Error: exec-all requires docker-compose arguments")
			fmt.Println("Usage: wpp-deployer exec-all [-r] <docker-compose-args...>")
			os.Exit(1)
		}

		if err := deployer.ExecAll(args, reloadNginx); err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}

	case "listen":
		port := "3000"
		secret := ""

		// Parse arguments for port and secret
		args := os.Args[2:]
		for i, arg := range args {
			switch arg {
			case "--port", "-p":
				if i+1 < len(args) {
					port = args[i+1]
				}
			case "--secret", "-s":
				if i+1 < len(args) {
					secret = args[i+1]
				}
			}
		}

		if err := deployer.Listen(port, secret); err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}

	case "add-repo":
		if len(os.Args) < 4 {
			fmt.Println("Error: add-repo requires repository and script")
			fmt.Println("Usage: wpp-deployer add-repo <username/repo> <script>")
			fmt.Println("Example: wpp-deployer add-repo myuser/myapp 'npm run build && wp plugin install \\$REPO_PATH/dist/plugin.zip --activate'")
			fmt.Println("Example: wpp-deployer add-repo myuser/site 'npm run build && wp theme install \\$REPO_PATH/dist/theme.zip --activate && wp db import \\$REPO_PATH/data.sql'")
			os.Exit(1)
		}

		repo := os.Args[2]
		script := os.Args[3]

		if err := deployer.AddRepo(repo, script); err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}

	case "list":
		sites, err := deployer.List()
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}
		for _, sitename := range sites {
			fmt.Println(sitename)
		}

	case "list-repos":
		if err := deployer.ListRepos(); err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}

	case "help", "-h", "--help":
		printUsage()

	case "version", "-v", "--version":
		fmt.Printf("%s v%s\n", appName, version)

	default:
		fmt.Printf("Error: unknown command '%s'\n", command)
		printUsage()
		os.Exit(1)
	}
}
