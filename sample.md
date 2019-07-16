## Sample response
```json
{
  "data": {
    "viewer": {
      "login": "someUser",
      "repositories": {
        "totalCount": 32,
        "pageInfo": {
          "endCursor": "Y3Vyc29yOnYyOpHOBEnudz==",
          "hasNextPage": true
        },
        "nodes": [
          {
            "name": "pipeline-samples",
            "url": "https://github.com/someUser/pipeline-samples",
            "id": "MDEwOlJlcG9zaXRvcnk3MTk1NDAzOz==",
            "sshUrl": "git@github.com:someUser/pipeline-samples.git",
            "owner": {
              "login": "someUser"
            },
            "repositoryTopics": {
              "nodes": [
                {
                  "topic": {
                    "name": "jenkins-pipeline"
                  }
                }
              ]
            },
            "refs": {
              "totalCount": 1,
              "nodes": [
                {
                  "name": "master",
                  "target": {
                    "committedDate": "2016-11-22T04:57:02Z"
                  }
                }
              ]
            }
          }
        ]
      }
    }
  }
}
```