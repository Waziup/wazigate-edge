FROM golang:1.13-alpine AS development

ENV CGO_ENABLED=0
ENV GO111MODULE=on

RUN apk add --no-cache ca-certificates tzdata git

COPY . /wazigate-edge
WORKDIR /wazigate-edge

RUN go build -ldflags "-s -w" -o build/wazigate-edge .

ENTRYPOINT ["tail", "-f", "/dev/null"]


################################################################################


# FROM alpine:latest AS production

# WORKDIR /root/

# RUN apk --no-cache add ca-certificates tzdata curl

# COPY wazigate-dashboard/node_modules/react/umd wazigate-dashboard/node_modules/react/umd
# COPY wazigate-dashboard/node_modules/react-dom/umd wazigate-dashboard/node_modules/react-dom/umd
# COPY wazigate-dashboard/index.html \
# #    wazigate-dashboard/dev.html \
#     wazigate-dashboard/favicon.ico \
#     wazigate-dashboard/wazigate.png \
#     wazigate-dashboard/site.webmanifest \
#     wazigate-dashboard/
# COPY wazigate-dashboard/dist wazigate-dashboard/dist
# 
# COPY --from=development /wazigate-edge/build/wazigate-edge .
# 
# ENTRYPOINT ["./wazigate-edge", "-www", "wazigate-dashboard"]