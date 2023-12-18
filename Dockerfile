# Use latest official Go image from the Docker Hub
FROM golang:1.21

ENV GO111MODULE=on

# Set the Current Working Directory inside the container
WORKDIR /app

# Copy everything from the current directory to the Working Directory inside the container
COPY src/ .

# Init go modules
RUN go mod init github.com/szibis/prometheus-governance-proxy && go mod tidy && go get

# Run unit tests
RUN go test -bench=. -benchtime=10s

# Build the Go app
RUN go build -gcflags '-l=4' -o main .

# This container exposes port 8080 to the outside world
EXPOSE 8080

# Run the binary program produced by `go build`
CMD ["./main"]
