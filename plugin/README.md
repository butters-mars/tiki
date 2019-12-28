### build the plugin for linux
1. Build the docker image for cross-compile
```
docker build -t builder:0.0.1 .
```
with Dockerfile
```
FROM golang:alpine
RUN apk add bash ca-certificates git gcc g++ libc-dev
```

2. Start the docker with -v YOUR_GOPATH/src:/go/src

3. Then inside the docker, build with ./build.sh
