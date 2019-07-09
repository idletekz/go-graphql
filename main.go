package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/machinebox/graphql"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
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

// T Note: struct fields must be public in order for unmarshal to
// correctly populate the data.
type T struct {
	AppID     string `yaml:"appID"`
	AppName   string `yaml:"appName"`
	Checkmarx struct {
		Team     string `yaml:"cx-team"`
		Instance string
		Enable   bool
	}
}

var token = os.Getenv("GITHUB_TOKEN")
var yesterday = time.Now().AddDate(0, 0, -1)
var rawMap = map[string]string{
	"github.com": "https://raw.githubusercontent.com",
}

func main() {
	var repos []*Repo
	client := graphql.NewClient("https://api.github.com/graphql")
	// client.Log = func(s string) { log.Println(s) }
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

	for _, commit := range repos {
		fmt.Printf("%#v\n", commit)
		ymlURL := fmt.Sprintf("%s/%s/%s", rawURL(commit.URL), commit.Branch, "props.yml")
		data, err := fetchWithToken(ymlURL)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(data)
		t := T{}
		err = yaml.Unmarshal([]byte(data), &t)
		if err != nil {
			log.Fatalf("error: %v", err)
		}
		fmt.Printf("--- t:\n%v\n\n", t)

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

// https://raw.githubusercontent.com/[USER-NAME]/[REPOSITORY-NAME]/[BRANCH-NAME]/[FILE-PATH]
func rawURL(url string) string {
	s := strings.Split(url, "/")
	return fmt.Sprintf("%s/%s/%s", rawMap[s[2]], s[3], s[4])
}

func fetchWithToken(url string) (string, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Add("Authorization", "Bearer "+token)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel() // releases resources if operation completes before timeout elapses
	req = req.WithContext(ctx)
	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()
	fmt.Printf("%v %[1]T", res.StatusCode)
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return "", fmt.Errorf("fetchWithToken status code: %v", res.StatusCode)
	}
	bodyText, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return "", fmt.Errorf("fetchWithToken %v", err)
	}
	s := string(bodyText)
	return s, nil
}
