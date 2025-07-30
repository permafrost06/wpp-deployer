package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

type GitHubPayload struct {
	Action      string      `json:"action"`
	Number      int         `json:"number"`
	PullRequest PullRequest `json:"pull_request"`
	Repository  Repository  `json:"repository"`
	Ref         string      `json:"ref"`
	Before      string      `json:"before"`
	After       string      `json:"after"`
}

type PullRequest struct {
	Number  int    `json:"number"`
	Title   string `json:"title"`
	State   string `json:"state"`
	Head    Branch `json:"head"`
	Base    Branch `json:"base"`
	HTMLURL string `json:"html_url"`
}

type Branch struct {
	Ref  string     `json:"ref"`
	SHA  string     `json:"sha"`
	Repo Repository `json:"repo"`
}

type Repository struct {
	Name     string `json:"name"`
	FullName string `json:"full_name"`
	HTMLURL  string `json:"html_url"`
	CloneURL string `json:"clone_url"`
}

type WebhookServer struct {
	port     string
	secret   string
	deployer *WPPDeployer
}

func NewWebhookServer(port, secret string, deployer *WPPDeployer) *WebhookServer {
	return &WebhookServer{
		port:     port,
		secret:   secret,
		deployer: deployer,
	}
}

func (ws *WebhookServer) validateSignature(payload []byte, signature string) bool {
	if ws.secret == "" {
		return true
	}

	if !strings.HasPrefix(signature, "sha256=") {
		return false
	}

	expectedMAC := hmac.New(sha256.New, []byte(ws.secret))
	expectedMAC.Write(payload)
	expectedSignature := "sha256=" + hex.EncodeToString(expectedMAC.Sum(nil))

	return hmac.Equal([]byte(signature), []byte(expectedSignature))
}

func (ws *WebhookServer) handleWebhook(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("Error reading request body: %v", err)
		http.Error(w, "Error reading request body", http.StatusBadRequest)
		return
	}

	signature := r.Header.Get("X-Hub-Signature-256")
	if !ws.validateSignature(body, signature) {
		log.Printf("Invalid signature for webhook request")
		http.Error(w, "Invalid signature", http.StatusUnauthorized)
		return
	}

	eventType := r.Header.Get("X-GitHub-Event")
	if eventType == "" {
		log.Printf("No GitHub event type in headers")
		http.Error(w, "No event type", http.StatusBadRequest)
		return
	}

	var payload GitHubPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		log.Printf("Error parsing JSON payload: %v", err)
		http.Error(w, "Error parsing JSON", http.StatusBadRequest)
		return
	}

	ws.handleGitHubEvent(eventType, &payload)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func (ws *WebhookServer) handleGitHubEvent(eventType string, payload *GitHubPayload) {
	repo := payload.Repository.FullName
	timestamp := time.Now().Format("15:04:05")

	switch eventType {
	case "pull_request":
		action := payload.Action
		pr := payload.PullRequest

		fmt.Printf("[%s] 🔀 PR #%d %s: %s\n", timestamp, pr.Number, action, pr.Title)
		fmt.Printf("         Repository: %s\n", repo)
		fmt.Printf("         Branch: %s → %s\n", pr.Head.Ref, pr.Base.Ref)
		fmt.Printf("         URL: %s\n", pr.HTMLURL)
		fmt.Println()

		if action == "opened" || action == "synchronize" {
			ws.handleRepositoryDeployment(repo, pr.Head.Ref, payload.Repository.CloneURL)
		}

	case "push":
		ref := payload.Ref
		branch := strings.TrimPrefix(ref, "refs/heads/")

		if !strings.HasPrefix(ref, "refs/heads/") {
			return
		}

		fmt.Printf("[%s] 📤 Push to %s\n", timestamp, repo)
		fmt.Printf("         Branch: %s\n", branch)
		fmt.Printf("         Commits: %s...%s\n", payload.Before[:8], payload.After[:8])
		fmt.Println()

		ws.handleRepositoryDeployment(repo, branch, payload.Repository.CloneURL)

	case "ping":
		fmt.Printf("[%s] 🏓 Webhook ping from %s\n", timestamp, repo)
		fmt.Println("         Webhook successfully configured!")
		fmt.Println()

	default:
		fmt.Printf("[%s] 📋 GitHub event: %s from %s\n", timestamp, eventType, repo)
		if payload.Action != "" {
			fmt.Printf("         Action: %s\n", payload.Action)
		}
		fmt.Println()
	}
}

func (ws *WebhookServer) handleRepositoryDeployment(repoFullName, branch, cloneURL string) {
	configs, err := ws.deployer.loadRepoConfigs()
	if err != nil {
		fmt.Printf("         [!] Error loading repo configs: %v\n", err)
		return
	}

	repoConfig, exists := configs[repoFullName]
	if !exists {
		fmt.Printf("         [!] Repository %s not configured for deployment\n", repoFullName)
		return
	}

	fmt.Printf("         [+] Repository configured for deployment\n")
	fmt.Printf("         [+] Build: %s\n", repoConfig.BuildCommand)
	fmt.Printf("         [+] Zip: %s\n", repoConfig.ZipLocation)

	siteName := ws.createSiteName(repoFullName, branch)
	fmt.Printf("         [+] Creating site: %s.nshlog.com\n", siteName)

	if err := ws.deployRepository(siteName, repoFullName, branch, cloneURL, repoConfig); err != nil {
		fmt.Printf("         [!] Deployment failed: %v\n", err)
		return
	}

	fmt.Printf("         [✔] Deployment completed successfully!\n")
	fmt.Println()
}

func (ws *WebhookServer) createSiteName(repoFullName, branch string) string {
	parts := strings.Split(repoFullName, "/")
	if len(parts) != 2 {
		return strings.ReplaceAll(repoFullName, "/", "-") + "-" + branch
	}

	username := parts[0]
	repoName := parts[1]

	cleanBranch := strings.ReplaceAll(branch, "/", "-")
	cleanBranch = strings.ReplaceAll(cleanBranch, "_", "-")

	return fmt.Sprintf("%s-%s-%s", username, repoName, cleanBranch)
}

func (ws *WebhookServer) deployRepository(siteName, repoFullName, branch, cloneURL string, config RepoConfig) error {
	workDir := ws.deployer.workDir
	repoDir := filepath.Join(workDir, "repos", repoFullName)

	if err := os.MkdirAll(filepath.Dir(repoDir), 0755); err != nil {
		return fmt.Errorf("failed to create repos directory: %w", err)
	}

	if err := ws.cloneOrUpdateRepo(repoDir, cloneURL, branch); err != nil {
		return fmt.Errorf("failed to clone/update repository: %w", err)
	}

	if err := ws.createOrUpdateSite(siteName); err != nil {
		return fmt.Errorf("failed to create/update site: %w", err)
	}

	if err := ws.runBuildCommand(repoDir, config.BuildCommand); err != nil {
		return fmt.Errorf("failed to run build command: %w", err)
	}

	if err := ws.installPlugin(repoDir, config.ZipLocation, siteName); err != nil {
		return fmt.Errorf("failed to install plugin: %w", err)
	}

	return nil
}

func (ws *WebhookServer) cloneOrUpdateRepo(repoDir, cloneURL, branch string) error {
	fmt.Printf("         [+] Updating repository...\n")

	if _, err := os.Stat(repoDir); os.IsNotExist(err) {
		fmt.Printf("         [+] Cloning repository...\n")
		cmd := exec.Command("git", "clone", "--depth", "1", "--branch", branch, cloneURL, repoDir)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("git clone failed: %w", err)
		}
	} else {
		fmt.Printf("         [+] Pulling latest changes...\n")

		cmd := exec.Command("git", "reset", "--hard")
		cmd.Dir = repoDir
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("git reset failed: %w", err)
		}

		cmd = exec.Command("git", "checkout", branch)
		cmd.Dir = repoDir
		if err := cmd.Run(); err != nil {
			cmd = exec.Command("git", "checkout", "-b", branch, "origin/"+branch)
			cmd.Dir = repoDir
			if err := cmd.Run(); err != nil {
				return fmt.Errorf("git checkout failed: %w", err)
			}
		}

		cmd = exec.Command("git", "pull", "origin", branch)
		cmd.Dir = repoDir
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("git pull failed: %w", err)
		}
	}

	return nil
}

func (ws *WebhookServer) createOrUpdateSite(siteName string) error {
	fmt.Printf("         [+] Setting up WordPress site...\n")

	sites, err := ws.deployer.List()
	if err != nil {
		return fmt.Errorf("failed to list sites: %w", err)
	}

	siteExists := false
	for _, site := range sites {
		if site == siteName {
			siteExists = true
			break
		}
	}

	if !siteExists {
		fmt.Printf("         [+] Creating new WordPress site: %s\n", siteName)
		if err := ws.deployer.Deploy(siteName); err != nil {
			return fmt.Errorf("failed to deploy WordPress site: %w", err)
		}
	} else {
		fmt.Printf("         [+] WordPress site already exists: %s\n", siteName)
	}

	return nil
}

func (ws *WebhookServer) runBuildCommand(repoDir, buildCommand string) error {
	fmt.Printf("         [+] Running build command: %s\n", buildCommand)

	cmd := exec.Command("bash", "-c", buildCommand)
	cmd.Dir = repoDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("build command failed: %w", err)
	}

	return nil
}

func (ws *WebhookServer) installPlugin(repoDir, zipLocation, siteName string) error {
	fmt.Printf("         [+] Installing WordPress plugin...\n")

	zipPath := filepath.Join(repoDir, zipLocation)

	if _, err := os.Stat(zipPath); os.IsNotExist(err) {
		return fmt.Errorf("plugin zip file not found: %s", zipPath)
	}

	absZipPath, err := filepath.Abs(zipPath)
	if err != nil {
		return fmt.Errorf("failed to get absolute path for zip: %w", err)
	}

	siteDir := filepath.Join(ws.deployer.workDir, fmt.Sprintf("wordpress-%s", siteName))
	dockerComposePath := filepath.Join(siteDir, "docker-compose.yml")

	fmt.Printf("         [+] Installing plugin: %s\n", zipLocation)
	cmd := exec.Command("docker", "compose", "-f", dockerComposePath, "run", "--rm", "wpcli", "plugin", "install", absZipPath, "--activate")
	cmd.Dir = siteDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("WP-CLI plugin install failed: %w", err)
	}

	fmt.Printf("         [+] Plugin installed and activated successfully!\n")
	return nil
}

func (ws *WebhookServer) healthCheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "healthy",
		"service": "wpp-deployer-webhook",
	})
}

func (ws *WebhookServer) Start() error {
	mux := http.NewServeMux()

	mux.HandleFunc("/webhook", ws.handleWebhook)

	mux.HandleFunc("/health", ws.healthCheck)

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/plain")
		fmt.Fprintf(w, "wpp-deployer webhook server\nEndpoints:\n  POST /webhook - GitHub webhooks\n  GET /health - Health check\n")
	})

	server := &http.Server{
		Addr:    ":" + ws.port,
		Handler: mux,
	}

	fmt.Printf("🚀 Webhook server starting on port %s\n", ws.port)
	fmt.Println("📋 Endpoints:")
	fmt.Printf("    POST http://localhost:%s/webhook - GitHub webhooks\n", ws.port)
	fmt.Printf("    GET  http://localhost:%s/health  - Health check\n", ws.port)
	fmt.Println()
	fmt.Println("🔗 Configure GitHub webhook URL: http://your-domain.com/webhook")
	fmt.Println("📝 Listening for GitHub events...")
	fmt.Println()

	return server.ListenAndServe()
}

func (w *WPPDeployer) Listen(port, secret string) error {
	if port == "" {
		port = "3000"
	}

	server := NewWebhookServer(port, secret, w)
	return server.Start()
}
