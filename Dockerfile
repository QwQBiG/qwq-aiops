FROM golang:1.23-alpine AS builder
WORKDIR /app
ENV GOPROXY=https://goproxy.cn,direct
COPY . .
RUN go mod tidy
RUN go build -o qwq cmd/qwq/main.go
FROM alpine:latest
WORKDIR /root/
RUN apk add --no-cache bash curl grep procps coreutils net-tools iproute2 tzdata tini
ENV TZ=Asia/Shanghai
COPY --from=builder /app/qwq .
EXPOSE 8899
ENTRYPOINT ["/sbin/tini", "--", "./qwq"]
CMD ["web"]