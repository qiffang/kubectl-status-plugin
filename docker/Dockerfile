FROM golang:1.15.8 as base

# this fils is generate by go build
COPY checker  /usr/local/bin/checker
ENTRYPOINT ["/usr/local/bin/checker", "run"]