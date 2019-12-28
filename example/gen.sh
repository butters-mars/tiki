#!/bin/sh
DIR=$1
OUT=$2

if [ "" == "$DIR" ]; then
    echo "dir is empty"
    exit
fi

if [ "" == "$OUT" ]; then
    echo "out is empty"
    exit
fi

echo "cleanup $OUT ..."
rm $OUT/*.pb.go
rm $OUT/*.pb.gw.go
rm $OUT/*.swagger.json

echo "generating *.pb.go ..."
protoc -I/usr/local/include -I$DIR \
  -I$GOPATH/src \
  -I$GOPATH/src/github.com/envoyproxy/protoc-gen-validate \
  -I$GOPATH/src/github.com/grpc-ecosystem/grpc-gateway/third_party/googleapis \
  --go_out=plugins=grpc:$OUT \
  --validate_out=lang=go:$OUT \
  $DIR/*.proto 

echo "generating reverse-proxy ..."
protoc -I/usr/local/include -I $DIR \
  -I$GOPATH/src \
  -I$GOPATH/src/github.com/grpc-ecosystem/grpc-gateway/third_party/googleapis \
  --grpc-gateway_out=logtostderr=true,grpc_api_configuration=$DIR/proxy.yaml:$OUT \
  $DIR/*.proto 

echo "generating swagger ..."
protoc -I/usr/local/include -I $DIR \
    -I$GOPATH/src \
    -I$GOPATH/src/github.com/grpc-ecosystem/grpc-gateway/third_party/googleapis \
    --swagger_out=logtostderr=true,grpc_api_configuration=$DIR/proxy.yaml:$OUT \
    $DIR/*.proto 

echo "go get ."
D=`pwd`
cd $OUT
go get .
cd $D