package main

import (
	"bytes"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"text/template"
	"time"
)

type ComposeParams struct {
	SiteName   string
	DBPassword string
	WPPort     int
}

func deployAndInstallPlugin(sitename, zipPath string) error {
	dbPass := randomString(16)
	wpPort := randomPort()
	params := ComposeParams{
		SiteName:   sitename,
		DBPassword: dbPass,
		WPPort:     wpPort,
	}

	composeFile, err := generateComposeFile(params)
	if err != nil {
		return fmt.Errorf("compose file: %w", err)
	}
	defer os.Remove(composeFile)

	// docker-compose up -d
	cmd := exec.Command("docker-compose", "-f", composeFile, "up", "-d")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("docker-compose up: %w", err)
	}

	// Wait for WordPress to be ready
	if err := waitForWP(wpPort); err != nil {
		return fmt.Errorf("wp not ready: %w", err)
	}

	// Copy plugin zip into container
	container := fmt.Sprintf("%s_wp", sitename)
	pluginDest := "/tmp/plugin.zip"
	cmd = exec.Command("docker", "cp", zipPath, fmt.Sprintf("%s:%s", container, pluginDest))
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("docker cp: %w", err)
	}

	// Install and activate plugin via WP-CLI
	cmd = exec.Command("docker", "exec", container, "wp", "plugin", "install", pluginDest, "--activate", "--allow-root")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("wp plugin install: %w", err)
	}

	return nil
}

func generateComposeFile(params ComposeParams) (string, error) {
	tmplBytes, err := os.ReadFile("docker-compose.tmpl.yml")
	if err != nil {
		return "", err
	}
	tmpl, err := template.New("compose").Parse(string(tmplBytes))
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, map[string]string{
		"SITE_NAME":   params.SiteName,
		"DB_PASSWORD": params.DBPassword,
		"WP_PORT":     fmt.Sprintf("%d", params.WPPort),
	}); err != nil {
		return "", err
	}
	fname := filepath.Join(os.TempDir(), fmt.Sprintf("docker-compose-%s.yml", params.SiteName))
	if err := os.WriteFile(fname, buf.Bytes(), 0644); err != nil {
		return "", err
	}
	return fname, nil
}

func waitForWP(port int) error {
	url := fmt.Sprintf("http://localhost:%d/wp-login.php", port)
	for i := 0; i < 30; i++ {
		resp, err := httpGet(url)
		if err == nil && resp == 200 {
			return nil
		}
		time.Sleep(4 * time.Second)
	}
	return fmt.Errorf("WordPress not ready on port %d", port)
}

func httpGet(url string) (int, error) {
	resp, err := http.Get(url)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	return resp.StatusCode, nil
}

func randomString(n int) string {
	letters := []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")
	s := make([]rune, n)
	for i := range s {
		s[i] = letters[rand.Intn(len(letters))]
	}
	return string(s)
}

func randomPort() int {
	return 20000 + rand.Intn(10000)
}
