FROM golang:1.23

WORKDIR /code
COPY ./go.mod ./go.sum ./
RUN go mod download

COPY . .
RUN go build -o main .

EXPOSE 2000

CMD [ "./main" ]
