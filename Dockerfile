FROM golang:1.21-alpine
WORKDIR /app
COPY . .
RUN go build .
EXPOSE 8080
ENTRYPOINT ./fuse-chat -p 8080 -d www.example.com