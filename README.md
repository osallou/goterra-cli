# Goterra-cli

Command line tool too interact with Goterra

## Build

    go build -ldflags "-X main.Version=`git rev-parse --short HEAD`" -o goterra goterra-cli.go

## Usage

    goterra-cli -h
