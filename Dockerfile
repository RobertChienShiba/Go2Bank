# Build stage
FROM golang:1.23.6-alpine3.21 AS builder
WORKDIR /app
COPY . .
RUN go build -o main main.go

# Run stage
FROM alpine:3.19
WORKDIR /app
COPY --from=builder /app/main .
COPY app.env .
COPY start.sh .
COPY db/migration ./db/migration

RUN chmod +x ./start.sh 

EXPOSE 8080 
# CMD [ "/app/main" ]
# ENTRYPOINT [ "/app/start.sh" ]