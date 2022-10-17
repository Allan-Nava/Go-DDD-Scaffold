# Go Domain Driven Design Scaffold

Generate scaffold domain driven design project layout for Go.

The following is Go Domain Driven Design project layout scaffold generated:

```

├── Dockerfile
├── Makefile
├── README.md
├── cmd
│   └── main.go
├── config
│   └── config.go
├── database
│   └── db.go
├── utils
│    ├── api_messages.go
│    └── env.go
└── docker-compose.yml

```


## Installation

Download Scaffold by using:
```sh
$ go get -u github.com/Allan-Nava/Go-DDD-Scaffold
```

## Create a new project

1. Going to your new project folder:

```bash
# change to project directory
$ cd $GOPATH/src/path/to/project
``` 

2. Run `scaffold init`in the new project folder:


```bash
$ scaffold init
```