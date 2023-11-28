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
		fmt.Println(err)
		return
	}

	client := github.NewClient(&http.Client{Transport: itr})

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		payload, err := github.ValidatePayload(r, []byte("wppwebhooksecret"))
		if err != nil {
			fmt.Println(err)
			return
		}
		event, err := github.ParseWebHook(github.WebHookType(r), payload)
		if err != nil {
			fmt.Println(err)
			return
		}

		switch event := event.(type) {
		case *github.WorkflowRunEvent:
			func() {
				if *event.Action != "completed" {
					return
				}
				if *event.WorkflowRun.Status != "completed" || *event.WorkflowRun.Conclusion != "success" {
					return
				}

				fmt.Println("action completed successfully")

				workflow_id := *event.WorkflowRun.ID

				artifacts, _, err := client.Actions.ListArtifacts(context.Background(), *event.Repo.Owner.Login, *event.Repo.Name, nil)

				if err != nil {
					fmt.Println(err)
					return
				}

				for _, artifact := range artifacts.Artifacts {
					if *artifact.WorkflowRun.ID == workflow_id {
						url, _, err := client.Actions.DownloadArtifact(context.Background(), "permafrost06", "contacts-manager-wp", *artifact.ID, 0)

						if err != nil {
							fmt.Println(err)
							return
						}

						DownloadFile(*url, *artifact.Name+".zip")

						owner, repo, issue_num := *event.Repo.Owner.Login, *event.Repo.Name, *event.WorkflowRun.PullRequests[0].Number
						_, _, err = client.Issues.CreateComment(context.Background(),
							owner,
							repo,
							issue_num,
							&github.IssueComment{
								Body: github.String(":wave: Hello from wpp deploy!"),
							})
						if err != nil {
							fmt.Println(err)
							return
						}
						break
					}
				}
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
