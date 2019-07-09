graphql connection:
https://blog.apollographql.com/explaining-graphql-connections-c48b7c3d6976

## what's a ref
A ref is anything pointing to a commit, for example, branches (heads), tags, and remote branches. You should see heads, remotes, and tags in your .git/refs directory, assuming you have all three types of refs in your repository.

refs/heads/0.58 specifies a branch named 0.58. If you don't specify what namespace the ref is in, git will look in the default ones. This makes using only 0.58 conceivably ambiguous - you could have both a branch and a tag named 0.58.

- why name a list of edges a "connection"?
	- Connection represents an abstract concept
	- Edge represents an actual entity in our graph
	- Edges: list of node
	- a connection is a way to get all the nodes that are connected to another node in a specific way
	- repositories:edges:node (list of all connected repositories that i'm an owner or member)
	- for a repository:
		- refs:edges:node (list 100 branch commit)
			- name:
			- target: 

what's ...on?
	- if you're querying a field that returns an interface or union type, you will need to use inline fragments to access data on the underlying concrete type
		... on <Implementation> {
			field
		}

repositories.pageInfo.hasNexPage?
pageInfo.hasNextPage ?

https://blog.golang.org/context
https://blog.machinebox.io/a-graphql-client-library-for-go-5bffd0455878
https://www.youtube.com/watch?v=F8nrpe0XWRg&list=PLq2Nv-Sh8EbbIjQgDzapOFeVfv5bGOoPE&index=3&t=0s

curl -H "Authorization: bearer xxx" -X POST -d "{\"query\": \"query { viewer { login }}\"}" https://api.github.com/graphql

https://www.mongodb.com/blog/post/mongodb-go-driver-tutorial
https://docs.mongodb.com/ecosystem/drivers/go/?jmp=blog

## yaml
- https://rhnh.net/2011/01/31/yaml-tutorial/
