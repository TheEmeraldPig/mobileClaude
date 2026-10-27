# The Remote game server. Build from the remote/ folder:
#   docker build -f server/Dockerfile -t remote-server .
FROM golang:1.24-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY mmo ./mmo
COPY srv ./srv
COPY server ./server
RUN CGO_ENABLED=0 go build -trimpath -ldflags "-s -w" -o /remote-server ./server

FROM alpine:3.20
RUN adduser -D remote
USER remote
WORKDIR /home/remote
COPY --from=build /remote-server /usr/local/bin/remote-server
ENV REMOTE_DATA=/home/remote/data
EXPOSE 8080
CMD ["remote-server"]
