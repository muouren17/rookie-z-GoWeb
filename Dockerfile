FROM golang:1.26.8-alpine AS builder

ENV GOPROXY=https://goproxy.cn,direct

WORKDIR /src

# 单独下载依赖，便于复用构建缓存
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /webapp .

FROM alpine:3.23

RUN apk add --no-cache ca-certificates tzdata \
    && mkdir -p /app/logs

ENV TZ=Asia/Shanghai

WORKDIR /app

COPY --from=builder /webapp ./webapp
COPY --from=builder /src/models/create_table.sql ./models/create_table.sql

EXPOSE 8081

# ENTRYPOINT ["./webapp"]