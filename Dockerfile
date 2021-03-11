FROM golang:alpine

WORKDIR /app
COPY . /app
ENV GO111MODULE=on
RUN go build -o ./bin/opensocks ./main.go

ENTRYPOINT ["./bin/opensocks"]

