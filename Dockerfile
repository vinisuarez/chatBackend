FROM golang:1.16-alpine

WORKDIR /app
ENV GO111MODULE=on

COPY . .
RUN go mod download

COPY *.go ./

RUN go build -o /docker-chatServer

ARG port
EXPOSE $port

ENTRYPOINT ["/docker-chatServer"]  