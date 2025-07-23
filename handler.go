package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
)

type GitHubPushEvent struct {
	Ref        string `json:"ref"`
	Repository struct {
		CloneURL string `json:"clone_url"`
		Name     string `json:"name"`
		Owner    struct {
			Name string `json:"name"`
		} `json:"owner"`
	} `json:"repository"`
	After string `json:"after"`
}

type GitHubPREvent struct {
	Action      string `json:"action"`
	PullRequest struct {
		Number int `json:"number"`
		Head   struct {
			Ref  string `json:"ref"`
			SHA  string `json:"sha"`
			Repo struct {
				CloneURL string `json:"clone_url"`
				Name     string `json:"name"`
				Owner    struct {
					Login string `json:"login"`
				} `json:"owner"`
			} `json:"repo"`
		} `json:"head"`
	} `json:"pull_request"`
	Repository struct {
		Name  string `json:"name"`
		Owner struct {
			Login string `json:"login"`
		} `json:"owner"`
	} `json:"repository"`
}

func handlePushEvent(payload []byte) {
	var event GitHubPushEvent
	if err := json.Unmarshal(payload, &event); err != nil {
		log.Printf("Failed to parse push event: %v", err)
		return
	}
	branch := strings.TrimPrefix(event.Ref, "refs/heads/")
	if branch == "" {
		log.Printf("Not a branch push: %s", event.Ref)
		return
	}
	name := fmt.Sprintf("%s-%s", event.Repository.Name, branch)
	cloneAndDeploy(event.Repository.CloneURL, branch, name, event.After)
}

func handlePREvent(payload []byte) {
	var event GitHubPREvent
	if err := json.Unmarshal(payload, &event); err != nil {
		log.Printf("Failed to parse PR event: %v", err)
		return
	}
	if event.Action != "opened" && event.Action != "synchronize" && event.Action != "reopened" {
		log.Printf("Ignoring PR action: %s", event.Action)
		return
	}
	pr := event.PullRequest
	name := fmt.Sprintf("%s-pr-%d", event.Repository.Name, pr.Number)
	cloneAndDeploy(pr.Head.Repo.CloneURL, pr.Head.Ref, name, pr.Head.SHA)
}

func cloneAndDeploy(cloneURL, ref, sitename, sha string) {
	workdir := filepath.Join("/tmp", fmt.Sprintf("%s-%s", sitename, sha[:7]))
	log.Printf("Cloning %s@%s to %s", cloneURL, ref, workdir)
	_, err := git.PlainClone(workdir, false, &git.CloneOptions{
		URL:           cloneURL,
		ReferenceName: plumbing.ReferenceName("refs/heads/" + ref),
		SingleBranch:  true,
		Depth:         1,
	})
	if err != nil {
		log.Printf("Clone failed: %v", err)
		return
	}

	// Build plugin zip
	if !runBuildCommand(workdir) {
		log.Printf("Build failed for %s", workdir)
		return
	}

	// Find the zip file
	zipPath := findZipFile(workdir)
	if zipPath == "" {
		log.Printf("No zip file found in %s", workdir)
		return
	}

	// Deploy WordPress site and install plugin
	if err := deployAndInstallPlugin(sitename, zipPath); err != nil {
		log.Printf("Deployment failed: %v", err)
	}
}

func runBuildCommand(dir string) bool {
	cmd := exec.Command("npm", "run", "build:zip")
	cmd.Dir = dir
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	if err := cmd.Run(); err != nil {
		log.Printf("npm run build:zip failed: %v\n%s", err, out.String())
		// Try pnpm
		cmd = exec.Command("pnpm", "run", "build:zip")
		cmd.Dir = dir
		out.Reset()
		cmd.Stdout = &out
		cmd.Stderr = &out
		if err := cmd.Run(); err != nil {
			log.Printf("pnpm run build:zip failed: %v\n%s", err, out.String())
			return false
		}
	}
	return true
}

func findZipFile(dir string) string {
	files, err := os.ReadDir(dir)
	if err != nil {
		return ""
	}
	for _, f := range files {
		if strings.HasSuffix(f.Name(), ".zip") {
			return filepath.Join(dir, f.Name())
		}
	}
	return ""
}

// func deployAndInstallPlugin(sitename, zipPath string) error {
// 	// TODO: Implement Docker Compose up, wait for WP, copy zip, install via WP-CLI
// 	log.Printf("Would deploy site %s and install %s", sitename, zipPath)
// 	return nil
// }
