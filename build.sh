#!/bin/sh
#
echo "is building()"
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -a -o main .
mkdir -p /tmp/bin
pwd
ls -al
tar -czvf /tmp/bin/go-scaffold.tar.gz --transform='s|.*/||'  ./main 
echo "builded()"
#tar --ignore-zeros -xf
ls -l /tmp/bin
#