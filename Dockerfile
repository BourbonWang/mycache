FROM golang:1.15.2

WORKDIR /go/src/cache
COPY . .
RUN export GO111MODULE=on
RUN go env -w GOPROXY=https://mirrors.aliyun.com/goproxy/

RUN go build -o server
RUN mv ./server /