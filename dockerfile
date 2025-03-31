FROM golang:1.24 AS builder
WORKDIR /code
COPY ./go.mod ./go.sum ./
RUN go mod download
COPY ./src ./src
RUN CGO_ENABLED=0 go build -o ./main ./src

FROM node:latest AS tailwind
WORKDIR /tw
RUN npm install tailwindcss @tailwindcss/cli
COPY ./html ./html
RUN npx @tailwindcss/cli -i ./html/tailwind.css -o ./tailwind.css --minify

FROM alpine:latest
WORKDIR /code
COPY ./assets ./assets
COPY ./html ./html
COPY ./static ./static
COPY --from=builder /code/main ./main
COPY --from=tailwind /tw/tailwind.css ./static/tailwind.css
EXPOSE 2000

CMD ["./main"]
