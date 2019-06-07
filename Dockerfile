FROM golang:1.12-alpine AS development

ENV PROJECT_PATH=/wazigateway-edge
ENV PATH=$PATH:$PROJECT_PATH/build
ENV CGO_ENABLED=0

RUN apk add --no-cache ca-certificates tzdata make git bash

RUN mkdir -p $PROJECT_PATH
COPY . $PROJECT_PATH
WORKDIR $PROJECT_PATH

RUN go build -o build/wazigateway-edge .

FROM alpine:latest AS production

WORKDIR /root/
RUN apk --no-cache add ca-certificates tzdata
COPY --from=development /wazigateway-edge/build/wazigateway-edge .
ENTRYPOINT ["./wazigateway-edge"]
