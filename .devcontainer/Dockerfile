FROM golang:1.23-bookworm

RUN apt update && apt install -y fuse nano file
RUN go install -v github.com/go-delve/delve/cmd/dlv@latest
