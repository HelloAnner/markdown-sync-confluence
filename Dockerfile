FROM golang:1.24.2-alpine

WORKDIR /app

COPY . .
RUN cd cmd/web && go build -o main

EXPOSE 8080

CMD ["./cmd/web/main"]