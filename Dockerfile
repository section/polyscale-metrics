# syntax=docker/dockerfile:1

FROM golang:1.19.3-alpine

WORKDIR /app

COPY go.mod ./
COPY go.sum ./
RUN go mod download

COPY *.go ./

RUN go build -o /polyscale-metrics

EXPOSE 8080

CMD [ "/polyscale-metrics" ]