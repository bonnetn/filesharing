FROM golang:1.16 AS builder

WORKDIR /go/src/github.com/bonnetn/filesharing

COPY ./go.mod ./
COPY ./main.go ./
COPY ./internal ./internal

RUN GOOS=linux CGO_ENABLED=1 go build .

FROM alpine:latest
EXPOSE 8080/tcp
RUN apk --no-cache add ca-certificates libc6-compat
WORKDIR /root/
COPY ./favicon.ico ./
COPY ./index.html ./
COPY --from=builder /go/src/github.com/bonnetn/filesharing/filesharing .
CMD ["./filesharing"]
