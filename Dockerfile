FROM golang:1.24-alpine

WORKDIR /app

RUN apk add --no-cache git build-base

COPY go.mod go.sum ./
RUN go mod download

COPY . .

ENV CGO_ENABLED=0
ARG TARGETOS
ARG TARGETARCH
RUN cd cmd/download && GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -o main

EXPOSE 8080

CMD ["./cmd/download/main"]