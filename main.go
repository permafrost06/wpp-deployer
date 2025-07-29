package main

import (
	"bufio"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strings"
	"text/template"
)

const (
	appName = "wpp-deployer"
	version = "1.0.0"
)

const (
	dockerComposeTemplateFile = "docker-compose.yml.template"
	nginxConfigTemplateFile   = "nginx-config.conf.template"
	nginxDockerComposeFile    = "nginx-docker-compose.yml.template"
	mainNginxConfigFile       = "wpp-deployer.conf.template"
	indexHTMLFile             = "index.html.template"
)

type TemplateData struct {
	Sitename string
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

	files := map[string]string{
		"nginx-docker-compose.yml":       nginxDockerComposeContent,
		"html/index.html":                indexHTMLContent,
		"nginx-config/wpp-deployer.conf": mainNginxConfigContent,
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

	tmplData := TemplateData{Sitename: sitename}

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
		return fmt.Errorf("failed to delete site directory: %w", err)
	}

	fmt.Printf("[✔] Site '%s' deleted successfully.\n", sitename)
	return nil
}

func (w *WPPDeployer) Control(command string, sitename string, includeVolumes bool) error {
	if command != "up" && command != "down" {
		return fmt.Errorf("invalid command: %s (use 'up' or 'down')", command)
	}

	nginxChanged := false

	if sitename != "" {
		if err := w.controlSite(command, sitename, includeVolumes, &nginxChanged); err != nil {
			return err
		}
	} else {
		fmt.Println("[*] No sitename specified, scanning all sites...")

		err := filepath.WalkDir(w.workDir, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}

			if d.IsDir() && strings.HasPrefix(d.Name(), "wordpress-") {
				dockerComposePath := filepath.Join(path, "docker-compose.yml")
				if _, err := os.Stat(dockerComposePath); err == nil {
					siteName := strings.TrimPrefix(d.Name(), "wordpress-")
					return w.controlSite(command, siteName, includeVolumes, &nginxChanged)
				}
			}
			return nil
		})

		if err != nil {
			return err
		}
	}

	if nginxChanged {
		if err := w.reloadNginx(); err != nil {
			return fmt.Errorf("failed to reload nginx: %w", err)
		}
	}

	return nil
}

func (w *WPPDeployer) controlSite(command, sitename string, includeVolumes bool, nginxChanged *bool) error {
	targetDir := filepath.Join(w.workDir, fmt.Sprintf("wordpress-%s", sitename))
	dockerComposePath := filepath.Join(targetDir, "docker-compose.yml")
	nginxConfig := filepath.Join(w.workDir, "nginx-config", fmt.Sprintf("%s.conf", sitename))
	nginxDisabled := nginxConfig + ".disabled"

	if _, err := os.Stat(targetDir); os.IsNotExist(err) {
		fmt.Printf("[!] Site directory 'wordpress-%s' not found\n", sitename)
		return nil
	}

	switch command {
	case "up":
		fmt.Printf("[↑] Bringing up wordpress-%s...\n", sitename)
		cmd := exec.Command("docker", "compose", "-f", dockerComposePath, "up", "-d")
		cmd.Dir = targetDir
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to bring up site %s: %w", sitename, err)
		}

		if _, err := os.Stat(nginxDisabled); err == nil {
			if err := os.Rename(nginxDisabled, nginxConfig); err != nil {
				return fmt.Errorf("failed to enable nginx config: %w", err)
			}
			fmt.Printf("[+] Enabled nginx config for '%s'\n", sitename)
			*nginxChanged = true
		}

	case "down":
		fmt.Printf("[↓] Bringing down wordpress-%s...\n", sitename)

		args := []string{"compose", "-f", dockerComposePath, "down"}
		if includeVolumes {
			args = append(args, "-v")
		}

		cmd := exec.Command("docker", args...)
		cmd.Dir = targetDir
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to bring down site %s: %w", sitename, err)
		}

		if _, err := os.Stat(nginxConfig); err == nil {
			if err := os.Rename(nginxConfig, nginxDisabled); err != nil {
				return fmt.Errorf("failed to disable nginx config: %w", err)
			}
			fmt.Printf("[−] Disabled nginx config for '%s'\n", sitename)
			*nginxChanged = true
		}
	}

	return nil
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
	cmd := exec.Command("docker", "exec", "wpp-deployer-nginx", "nginx", "-t")
	if err := cmd.Run(); err != nil {
		fmt.Println("[!] Skipped nginx reload: config test failed")
		return nil
	}

	cmd = exec.Command("docker", "exec", "wpp-deployer-nginx", "nginx", "-s", "reload")
	return cmd.Run()
}

func printUsage() {
	fmt.Printf(`%s v%s - WordPress Site Deployer

Usage:
  %s <command> [options] [arguments]

Commands:
  install               Set up %s workspace in ~/.%s
  deploy <sitename>     Deploy a new WordPress site
  delete <sitename>     Delete an existing WordPress site
  up [sitename]         Start site(s) - all sites if no sitename provided
  down [options] [sitename]  Stop site(s) - all sites if no sitename provided

Options for 'down':
  -v, --volumes        Remove volumes and disable nginx config

Examples:
  %s install
  %s deploy mysite
  %s delete mysite
  %s up mysite
  %s down -v mysite
  %s up
  %s down

`, appName, version, appName, appName, appName, appName, appName, appName, appName, appName, appName, appName)
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

	case "up":
		sitename := ""
		if len(os.Args) >= 3 {
			sitename = os.Args[2]
		}
		if err := deployer.Control("up", sitename, false); err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}

	case "down":
		downFlags := flag.NewFlagSet("down", flag.ExitOnError)
		volumes := downFlags.Bool("v", false, "Remove volumes and disable nginx config")
		volumesLong := downFlags.Bool("volumes", false, "Remove volumes and disable nginx config")

		args := os.Args[2:]
		downFlags.Parse(args)

		includeVolumes := *volumes || *volumesLong

		sitename := ""
		if len(downFlags.Args()) > 0 {
			sitename = downFlags.Args()[0]
		}

		if err := deployer.Control("down", sitename, includeVolumes); err != nil {
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
