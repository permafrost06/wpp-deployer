package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"

	"github.com/bradleyfalzon/ghinstallation"
	"github.com/google/go-github/v56/github"
)

func main() {
	itr, err := ghinstallation.NewKeyFromFile(http.DefaultTransport, 647423, 44338794, "wpp-deployer.2023-11-23.private-key.pem")

	if err != nil {
		fmt.Println("Couldn't create app key", err)
		return
	}

	client := github.NewClient(&http.Client{Transport: itr})

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		payload, err := github.ValidatePayload(r, []byte("wppwebhooksecret"))
		if err != nil {
			fmt.Println("Could not validate payload", err)
			return
		}
		event, err := github.ParseWebHook(github.WebHookType(r), payload)
		if err != nil {
			fmt.Println("Could not parse webhook", err)
			return
		}

		getArtifact := func(workflow_id int64) {
			artifacts, _, err := client.Actions.ListArtifacts(context.Background(), *event.Repo.Owner.Login, *event.Repo.Name, nil)

			if err != nil {
				fmt.Println("Couldn't get artifacts", err)
				return
			}

			for _, artifact := range artifacts.Artifacts {
				if *artifact.WorkflowRun.ID == workflow_id {
					url, _, err := client.Actions.DownloadArtifact(context.Background(), "permafrost06", "contacts-manager-wp", *artifact.ID, 0)

					if err != nil {
						fmt.Println("Couldn't get artifact download url", err)
						return
					}

					err = DownloadFile(*url, *artifact.Name+".zip")

					if err != nil {
						fmt.Println("Couldn't download artifact", err)
					}

					break
				}
			}
		}

		postComment := func(owner string, repo string, issue_num int, body string) {
			_, _, err = client.Issues.CreateComment(context.Background(),
				owner,
				repo,
				issue_num,
				&github.IssueComment{
					Body: github.String(body),
				})
			if err != nil {
				fmt.Println("Could not post comment", err)
				return
			}
		}

		switch event := event.(type) {
		case *github.WorkflowRunEvent:
			fmt.Println("WorkflowRunEvent received")
			func() {
				if *event.Action != "completed" ||
					*event.WorkflowRun.Status != "completed" ||
					*event.WorkflowRun.Conclusion != "success" {
					fmt.Println("Workflow not completed or unsuccessful")
					return
				}

				fmt.Println("Action completed successfully")

				workflow_id := *event.WorkflowRun.ID

				fmt.Println("Getting artifact")
				getArtifact(workflow_id)
				postComment(*event.Repo.Owner.Login,
					*event.Repo.Name,
					*event.WorkflowRun.PullRequests[0].Number,
					":wave: Hello from wpp deploy!")
			}()
		}
	})

	http.ListenAndServe(":3000", nil)
}

func DownloadFile(url url.URL, filepath string) error {

	// Get the data
	resp, err := http.Get(url.String())
	if err != nil {

		return err
	}
	defer resp.Body.Close()

	// Create the file
	if filepath == "" {
		filepath = path.Base(resp.Request.URL.String())
	}
	out, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer out.Close()

	// Write the body to file
	_, err = io.Copy(out, resp.Body)
	return err
}
