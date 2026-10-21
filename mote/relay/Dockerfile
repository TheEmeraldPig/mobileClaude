# The Mote multiplayer relay. Build from the mote/ folder:
#   docker build -f relay/Dockerfile -t mote-relay .
FROM golang:1.24-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY relay ./relay
RUN CGO_ENABLED=0 go build -ldflags "-s -w" -o /mote-relay ./relay

FROM alpine:3.20
COPY --from=build /mote-relay /mote-relay
ENV PORT=8080
EXPOSE 8080
CMD ["/mote-relay"]
