#####################################
#   STEP 1 build executable binary  #
#####################################
FROM golang:alpine AS builder

# Install git.
# Git is required for fetching the dependencies.
RUN apk update && apk add --no-cache git

WORKDIR /app

ENV GO111MODULE=on

# COPY go.mod .
# COPY go.sum .
COPY . .

RUN go mod download


# Build the binary.
RUN CGO_ENABLED=0 GOOS=linux go build -o main ./cmd

#####################################
#   STEP 2 build a small image      #
#####################################
FROM golang:alpine
# FROM scratch

# Copy our static executable.
COPY --from=builder /app/main /app/main

# Run the hello binary.
ENTRYPOINT ["/app/main"]

