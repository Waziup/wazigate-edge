FROM golang:1.12-alpine AS development

ENV CGO_ENABLED=0

RUN apk add --no-cache ca-certificates tzdata git

# COPY . /wazigate-edge
# WORKDIR /wazigate-edge
WORKDIR /go/src/wazigate-edge

ENTRYPOINT ["tail", "-f", "/dev/null"]

# # WAZIGATE_BRANCH=$(git rev-parse --abbrev-ref HEAD)
# # WAZIGATE_VERSION=$(git describe --always);
# RUN go build -ldflags "-s -w -X main.version=$WAZIGATE_VERSION -X main.branch=$WAZIGATE_BRANCH" -o build/wazigate-edge .

# FROM alpine:latest AS production

# WORKDIR /root/
# RUN apk --no-cache add ca-certificates tzdata curl
# COPY --from=development /wazigate-edge/build/wazigate-edge .
# COPY www www/
# ENTRYPOINT ["./wazigate-edge", "-www", "www"]
