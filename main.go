package main

import (
	  "fmt"
	  "os"

	  "github.com/cli/go-gh/v2/pkg/api"
)

type Issue struct {
	Number int    `json:"number"`
	Title  string `json:"title"`
	State  string `json:"state"`
}




func main() {
    if len(os.Args) < 2 {
    	fmt.Fprintln(os.Stderr, "usage: ghtui <owner>/<repo>")
    	os.Exit(1)
    }
    repo := os.Args[1]

	client, err := api.DefaultRESTClient()
	if err != nil {
		fmt.Fprintln(os.Stderr, "error creating client:", err)
		os.Exit(1)
	}

	var issues []Issue
	err = client.Get(fmt.Sprintf("repos/%s/issues", repo), &issues)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error fetching issues:", err)
		os.Exit(1)
	}

	for _, issue := range issues {
		fmt.Printf("#%d [%s] %s\n", issue.Number, issue.State, issue.Title)
	}
}
