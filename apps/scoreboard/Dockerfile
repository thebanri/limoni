# Lemon Hunt's scoreboard, for Render or Railway, or anything that runs a
# container.
FROM golang:1.25 AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY *.go ./
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /scoreboard .

# Distroless and static: no shell, no package manager, one binary. It runs
# as root so that it can write to a volume a host mounts owned by root.
FROM gcr.io/distroless/static-debian12
COPY --from=build /scoreboard /scoreboard
ENTRYPOINT ["/scoreboard"]
