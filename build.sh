#!/bin/sh

# This script assumes you've run
#   go get -u github.com/jteeuwen/go-bindata/...
# at least once.

cd bin/journal-server
cd assets
sass --update --style=compressed scss:css
cd ..
go generate
cd ../..

go build -o jrnl  bin/jrnl/*.go
go build -o journal-server  bin/journal-server/*.go

