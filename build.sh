#!/bin/bash

TARGETS="\
windows/386
windows/amd64
darwin/386
darwin/amd64
linux/386
linux/amd64
linux/arm
freebsd/386
freebsd/amd64
freebsd/arm
netbsd/386
netbsd/amd64
netbsd/arm
openbsd/386
openbsd/amd64
plan9/386
plan9/amd64
solaris/amd64"

mkdir -p bin

echo -n "getting dependencies..."
go get
echo " done."

for TARGET in $TARGETS; do
    IFS='/' read GOOS GOARCH <<< "$TARGET"
    OUT="bin/ditaa-$GOOS-$GOARCH"
    if [ "$GOOS" == "windows" ]; then
        OUT="$OUT.exe"
    fi

    echo -n "building $OUT..."
    env GOOS=$GOOS GOARCH=$GOARCH go build -o $OUT
    echo " done."
done

