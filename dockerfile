FROM golang:latest AS builder

WORKDIR /code
COPY ./go.mod ./go.sum ./
RUN go mod download

COPY ./src ./src
RUN CGO_ENABLED=0 go build -o ./main ./src

FROM alpine:latest
WORKDIR /code
COPY --from=builder /code/main /code/main
COPY ./assets ./assets
COPY ./html ./html
COPY ./static ./static
EXPOSE 2000

CMD ["./main"]
