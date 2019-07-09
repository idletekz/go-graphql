package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/machinebox/graphql"
	"log"
	"os"
	"time"
)

const search = `
query { 
  viewer { 
    login
    repositories(first: 100, isFork:false, affiliations:[OWNER, ORGANIZATION_MEMBER]) {
    	totalCount
    	pageInfo {
    		endCursor
    		hasNextPage
    	}
    	nodes {
    		name
    		url
    		id
    		sshUrl
        repositoryTopics(first: 100) {
          nodes {
            topic {
              name
            }
          }
        }    		
    		refs(first: 100, refPrefix: "refs/heads/") {
    			totalCount
    			nodes {
    				name
    				target {
    					...on Commit {
    						committedDate
    					}
    				}
    			}
    		}
    	}
    }
  }
}`

const nextSearch = `
query($after :String!) { 
  viewer { 
    login
    repositories(first: 100, isFork:false,after:$after, affiliations:[OWNER, ORGANIZATION_MEMBER]) {
    	totalCount
    	pageInfo {
    		endCursor
    		hasNextPage
    	}
    	nodes {
    		name
    		url
    		id
    		sshUrl
        repositoryTopics(first: 100) {
        	totalCount
          nodes {
            topic {
              name
            }
          }
        }    		
    		refs(first: 100, refPrefix: "refs/heads/") {
    			totalCount
    			nodes {
    				name
    				target {
    					...on Commit {
    						committedDate
    					}
    				}
    			}
    		}
    	}
    }
  }
}`

// Response ...
type Response struct {
	Viewer struct {
		Login        string
		Repositories struct {
			TotalCount int
			PageInfo   struct {
				EndCursor   string
				HasNextPage bool
			}
			Nodes []*Repository
		}
	}
}

// Repository struct
type Repository struct {
	Name             string
	URL              string
	ID               string
	SSHURL           string
	RepositoryTopics struct {
		Nodes []struct {
			Topic struct {
				Name string
			}
		}
	}
	Refs struct {
		TotalCount int
		Nodes      []struct {
			Name   string
			Target struct {
				CommittedDate time.Time
			}
		}
	}
}

// Repo  ...
type Repo struct {
	Name   string
	URL    string
	SSHURL string
	Branch string
}

var yesterday = time.Now().AddDate(0, 0, -1)

func main() {
	var repos []*Repo
	client := graphql.NewClient("https://api.github.com/graphql")
	client.Log = func(s string) { log.Println(s) }
	token := os.Getenv("GITHUB_TOKEN")
	ctx := context.Background()
	req := graphql.NewRequest(search)
	req.Header.Add("Authorization", "Bearer "+token)
	var respData Response
	if err := client.Run(ctx, req, &respData); err != nil {
		log.Fatal(err)
	}
	repos = ActiveTopic(respData.Viewer.Repositories.Nodes, "go")
	for respData.Viewer.Repositories.PageInfo.HasNextPage {
		req := graphql.NewRequest(nextSearch)
		req.Header.Add("Authorization", "Bearer "+token)
		req.Var("after", respData.Viewer.Repositories.PageInfo.EndCursor)
		if err := client.Run(ctx, req, &respData); err != nil {
			log.Fatal(err)
		}
		tRepos := ActiveTopic(respData.Viewer.Repositories.Nodes, "go")
		repos = append(repos, tRepos...)
	}

	fmt.Println("\n")
	for _, commit := range repos {
		fmt.Printf("%#v\n", commit)
	}
}

func pp(respData *Response) {
	data, err := json.MarshalIndent(respData, "", "  ")
	if err != nil {
		log.Fatalf("JSON marshaling failed: %s", err)
	}
	fmt.Printf("%s\n", data)
}

// ActiveTopic collect active repositories with specified topic
func ActiveTopic(repositories []*Repository, topic string) (active []*Repo) {
	for _, repo := range repositories {
		for _, node := range repo.RepositoryTopics.Nodes {
			if node.Topic.Name == topic {
				for _, branch := range repo.Refs.Nodes {
					if branch.Target.CommittedDate.After(yesterday) {
						r := &Repo{
							Name:   repo.Name,
							Branch: branch.Name,
							SSHURL: repo.SSHURL,
							URL:    repo.URL,
						}
						active = append(active, r)
					}
				}
				break
			}
		}
	}
	return active
}
