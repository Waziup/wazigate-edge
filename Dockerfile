FROM golang:1.13-alpine AS development

# Please change this when a new build is going to be pushed
ENV EDGE_VERSION=2.1.11

ENV CGO_ENABLED=0
ENV GO111MODULE=on

RUN apk add --no-cache ca-certificates tzdata git curl \
    && echo $EDGE_VERSION > /ver.txt

COPY . /wazigate-edge
WORKDIR /wazigate-edge

RUN go build -ldflags "-s -w" -o build/wazigate-edge .

# just to make development a bit easier
WORKDIR /go/src/github.com/Waziup/wazigate-edge/
ENTRYPOINT ["tail", "-f", "/dev/null"]

#---------------------------------------#

# Uncomment the follwoing lines for production:

FROM alpine:latest AS production

WORKDIR /root/

RUN apk --no-cache add ca-certificates tzdata curl

COPY wazigate-dashboard/node_modules/react/umd wazigate-dashboard/node_modules/react/umd
COPY wazigate-dashboard/node_modules/react-dom/umd wazigate-dashboard/node_modules/react-dom/umd
COPY wazigate-dashboard/index.html \
    #    wazigate-dashboard/dev.html \
    wazigate-dashboard/favicon.ico \
    wazigate-dashboard/wazigate.png \
    wazigate-dashboard/site.webmanifest \
    wazigate-dashboard/
COPY wazigate-dashboard/dist wazigate-dashboard/dist
COPY wazigate-dashboard/docs wazigate-dashboard/docs
COPY wazigate-dashboard/admin wazigate-dashboard/admin

COPY --from=development /wazigate-edge/build/wazigate-edge .


COPY --from=development /ver.txt /
RUN echo "export EDGE_VERSION=$(cat /ver.txt);" > start.sh \
    && echo "./wazigate-edge -www wazigate-dashboard" >> start.sh \
    && chmod +x start.sh

ENTRYPOINT ["sh", "start.sh"]

# ENTRYPOINT ["./wazigate-edge", "-www", "wazigate-dashboard"]
