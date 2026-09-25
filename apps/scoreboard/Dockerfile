# Lemon Hunt's scoreboard, for Railway (or anything that runs a container).
FROM golang:1.25 AS build
WORKDIR /src
COPY go.mod ./
COPY *.go ./
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /scoreboard .

# Distroless and static: no shell, no package manager, a 6 MB binary. It runs
# as root so that it can write to the volume Railway mounts, which is owned
# by root.
FROM gcr.io/distroless/static-debian12
COPY --from=build /scoreboard /scoreboard
ENTRYPOINT ["/scoreboard"]
