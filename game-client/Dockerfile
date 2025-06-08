FROM golang:1.22
WORKDIR /app
COPY . .
RUN go mod tidy
RUN go mod vendor
RUN go build -o leaderboard-service
EXPOSE 8081
CMD ["./leaderboard-service"]