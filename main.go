package main

import (
	"context"
	"fmt"
	"net/http"

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
		case *github.PullRequestEvent:
			func() {
				if *event.Action != "opened" {
					return
				}

				owner, repo, number := *event.Repo.Owner.Login, *event.Repo.Name, *event.PullRequest.Number
				_, _, err = client.Issues.CreateComment(context.Background(),
					owner,
					repo,
					number,
					&github.IssueComment{
						Body: github.String(":wave: Hello from wpp deploy!"),
					})
				if err != nil {
					fmt.Println(err)
					return
				}
			}()
		}
	})

	http.ListenAndServe(":3000", nil)
}
