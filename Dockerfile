FROM waziup/node-sass:14 AS dashboard

COPY wazigate-dashboard/. /wazigate-dashboard

WORKDIR /wazigate-dashboard/

RUN npm i && npm run build

################################################################################


FROM golang:1.13-alpine AS bin

ENV CGO_ENABLED=0
ENV GO111MODULE=on

RUN apk add --no-cache ca-certificates tzdata git

COPY . /wazigate-edge

WORKDIR /wazigate-edge/

RUN go build -ldflags "-s -w" -o wazigate-edge .

################################################################################


FROM alpine:latest AS app

RUN apk add --no-cache ca-certificates tzdata curl

WORKDIR /root/

COPY --from=dashboard /wazigate-dashboard/node_modules/react/umd wazigate-dashboard/node_modules/react/umd
COPY --from=dashboard /wazigate-dashboard/node_modules/react-dom/umd wazigate-dashboard/node_modules/react-dom/umd
COPY --from=dashboard /wazigate-dashboard/index.html \
    #    wazigate-dashboard/dev.html \
    /wazigate-dashboard/favicon.ico \
    /wazigate-dashboard/wazigate.png \
    /wazigate-dashboard/site.webmanifest \
    wazigate-dashboard/
COPY --from=dashboard /wazigate-dashboard/dist wazigate-dashboard/dist
COPY --from=dashboard /wazigate-dashboard/docs wazigate-dashboard/docs
COPY --from=dashboard /wazigate-dashboard/admin wazigate-dashboard/admin

COPY --from=bin /wazigate-edge/wazigate-edge .

EXPOSE 80/tcp
EXPOSE 1883/tcp

ENV WAZIUP_MONGO=wazigate-mongo:27017

HEALTHCHECK CMD curl --fail http://localhost || exit 1 

VOLUME /var/lib/wazigate

ENTRYPOINT ["./wazigate-edge", "-www", "wazigate-dashboard"]
