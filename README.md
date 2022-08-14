# Plugin Loading Issue

This repo contains a test of building a golang plugin and plugin loader in different gopath.
To sucessfully load the plugin and builder must be built with the linking parameter -trimpath

Relevant golang Issues:<br/>
[loading on machine with different GOPATH fails](https://github.com/golang/go/issues/26759)
<br/>
[support reproducible builds regardless of build path](https://github.com/golang/go/issues/16860)

Test building in alpine container:
```
docker run -it -v $(pwd):/src golang:1.18.1-alpine3.15 ash
cd /src

# Trying to build and load plugin
./test-plugin-loader.sh

# Result
# cgo: C compiler "gcc" not found: exec: "gcc": executable file not found in $PATH
# /src
# Building server
# go: missing Git command. See https://golang.org/s/gogetcmd
# error obtaining VCS status: exec: "git": executable file not found in $PATH
#         Use -buildvcs=false to disable VCS stamping.
# /src
# Running server
# ./test-plugin-loader.sh: line 30: ./server: not found

# install gcc and git
apk add build-base git

# Unsucessfully (without build argument -trimpath)
./test-plugin-loader.sh

# Sucessfully (with build argument -trimpath)
./test-plugin-loader.sh -trimpath

 ```
