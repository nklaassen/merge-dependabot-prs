package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"strings"

	"github.com/google/go-github/v48/github"
)

func main() {
	var (
		username string
	)
	flag.StringVar(&username, "username", "", "github username")
	flag.Parse()

	if username == "" {
		log.Fatal("must provide username")
	}

	if err := run(username); err != nil {
		log.Fatalf("%v", err)
	}
}

func run(username string) error {
	ctx := context.Background()
	client := github.NewClient(nil)

	query := fmt.Sprintf("repo:gravitational/teleport type:pr is:open author:app/dependabot assignee:%q", username)
	result, _, err := client.Search.Issues(ctx, query, &github.SearchOptions{
		ListOptions: github.ListOptions{
			PerPage: 30,
		},
	})
	if err != nil {
		return err
	}

	type PR struct {
		number int
		head   string
		url    string
	}
	var prs []PR

	for _, issue := range result.Issues {
		pr, _, err := client.PullRequests.Get(ctx, "gravitational", "teleport", *issue.Number)
		if err != nil {
			return err
		}
		prs = append(prs, PR{
			number: *pr.Number,
			head:   *pr.Head.SHA,
			url:    *pr.URL,
		})
	}

	var commitSHAs []string
	var prCloseCommands []string
	var dependabotPRs []string
	for _, pr := range prs {
		commitSHAs = append(commitSHAs, pr.head)
		prCloseCommands = append(prCloseCommands, fmt.Sprintf(`gh pr close %d --comment "closing in favor of #${PR}"`, pr.number))
		dependabotPRs = append(dependabotPRs, fmt.Sprintf("#%d", pr.number))
	}

	fmt.Println("copy, paste, and run each command output below while in the teleport repo directory")
	fmt.Println()
	fmt.Printf("> git switch -c %s/dependency-updates\n", username)
	fmt.Println()
	fmt.Println("> git cherry-pick ", strings.Join(commitSHAs, " "))
	fmt.Println()
	fmt.Println(`> go mod tidy && git add go.mod go.sum && pushd api && go mod tidy && git add go.mod go.sum && popd && git commit -m "go mod tidy"`)
	fmt.Println()
	fmt.Printf(`> gh pr create --web --title "Dependency updates" --body 'This PR replaces the following PRs opened by dependabot:

- %s
'
`,
		strings.Join(dependabotPRs, "\n- "))
	fmt.Println()
	fmt.Println("> export PR=#<number of new PR>")
	fmt.Println()
	fmt.Printf("> %s\n", strings.Join(prCloseCommands, " && \\\n"))

	return nil
}
