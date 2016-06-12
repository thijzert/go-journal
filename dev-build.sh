#!/bin/sh

# This script assumes you've run
#   go get -u github.com/jteeuwen/go-bindata/...
# at least once.


cd bin/journal-server
cd assets
sass --update --style=nested scss:css
cd ..
go-bindata -debug -o assets.go -pkg main assets/...
cd ../..

