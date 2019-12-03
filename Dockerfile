FROM golang:1.12-alpine AS development

ENV PROJECT_PATH=/wazigate-edge
ENV PATH=$PATH:$PROJECT_PATH/build
ENV CGO_ENABLED=0

RUN apk add --no-cache ca-certificates tzdata make git bash

RUN mkdir -p $PROJECT_PATH
COPY . $PROJECT_PATH
WORKDIR $PROJECT_PATH

RUN export branch=$(git rev-parse --abbrev-ref HEAD);
RUN export version=$(git describe --always);

RUN go build -ldflags "-s -w -X main.version=$version -X main.branch=$branch" -o build/wazigate-edge .

FROM alpine:latest AS production

WORKDIR /root/
RUN apk --no-cache add ca-certificates tzdata curl
COPY --from=development /wazigate-edge/build/wazigate-edge .
COPY www www/
ENTRYPOINT ["./wazigate-edge", "-www", "www"]
