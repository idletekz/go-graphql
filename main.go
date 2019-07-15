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
	"os/exec"
	"path/filepath"
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
    		owner {
    			login
    		}    		
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
    		owner {
    			login
    		}
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
	Name   string
	URL    string
	ID     string
	SSHURL string
	Owner  struct {
		Login string
	}
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
	Owner  string
}

// T Note: struct fields must be public in order for unmarshal to
// correctly populate the data.
type T struct {
	AppID   string `yaml:"appID"`
	AppName string `yaml:"appName"`
	Check   struct {
		Team     string `yaml:"team"`
		Instance string
		Enable   bool
	}
}

var token = os.Getenv("GITHUB_TOKEN")
var yesterday = time.Now().AddDate(0, 0, -1)
var rawContentURL = map[string]string{
	"github.com": "https://raw.githubusercontent.com",
}

func main() {
	repos, err := activities("go")
	if err != nil {
		log.Fatal(err)
	}
	for _, repo := range repos {
		fmt.Printf("%#v\n", repo)
		err := repo.clone()
		if err != nil {
			log.Fatal(err)
		}
		data, err := repo.raw("props.yml")
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
		repo.clone()
	}
}

func pp(respData *Response) {
	data, err := json.MarshalIndent(respData, "", "  ")
	if err != nil {
		log.Fatalf("JSON marshaling failed: %s", err)
	}
	fmt.Printf("%s\n", data)
}

func activities(topic string) (repos []*Repo, err error) {
	client := graphql.NewClient("https://api.github.com/graphql")
	// client.Log = func(s string) { log.Println(s) }
	ctx := context.Background()
	req := graphql.NewRequest(search)
	req.Header.Add("Authorization", "Bearer "+token)
	var respData Response
	if err = client.Run(ctx, req, &respData); err != nil {
		return nil, err
	}
	repos = activeTopic(respData.Viewer.Repositories.Nodes, topic)
	for respData.Viewer.Repositories.PageInfo.HasNextPage {
		req := graphql.NewRequest(nextSearch)
		req.Header.Add("Authorization", "Bearer "+token)
		req.Var("after", respData.Viewer.Repositories.PageInfo.EndCursor)
		if err := client.Run(ctx, req, &respData); err != nil {
			return nil, err
		}
		tRepos := activeTopic(respData.Viewer.Repositories.Nodes, topic)
		repos = append(repos, tRepos...)
	}
	return
}

// ActiveTopic collect active repositories with specified topic
func activeTopic(repositories []*Repository, topic string) (active []*Repo) {
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
							Owner:  repo.Owner.Login,
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
func (r *Repo) rawURL() string {
	s := strings.Split(r.URL, "/")
	return fmt.Sprintf("%s/%s/%s", rawContentURL[s[2]], s[3], s[4])
}

func (r *Repo) raw(path string) (string, error) {
	url := fmt.Sprintf("%s/%s/%s", r.rawURL(), r.Branch, path)
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
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return "", fmt.Errorf("rawContent status code: %v", res.StatusCode)
	}
	bodyText, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return "", fmt.Errorf("rawContent %v", err)
	}
	s := string(bodyText)
	return s, nil
}

func (r *Repo) clone() error {
	s := strings.Split(r.URL, "//")
	repo := fmt.Sprintf("%s//%s@%s", s[0], token, s[1])
	args := []string{
		"clone",
		"--depth=1",
		"-b",
		r.Branch,
		repo,
	}
	dir, err := r.createCloneDir()
	if err != nil {
		return fmt.Errorf("clone: %s", err)
	}
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err = cmd.Run()
	if err != nil {
		return err
	}
	return nil
}

func (r *Repo) createCloneDir() (string, error) {
	var dir string
	pwd, err := os.Getwd()
	if err != nil {
		return dir, fmt.Errorf("createCloneDir getwd: %s", err)
	}
	dir = filepath.Join(pwd, r.Owner, r.Branch)
	err = os.MkdirAll(dir, 0755)
	if err != nil {
		return dir, fmt.Errorf("createCloneDir mkdirall: %s", err)
	}
	return dir, nil
}
