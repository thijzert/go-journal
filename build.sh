#!/bin/sh

cd bin/journal-server
cd assets
sass --update --style=compressed scss:css
cd ../../..

go build -o jrnl  bin/jrnl/*.go
exec go build -o journal-server  bin/journal-server/*.go

