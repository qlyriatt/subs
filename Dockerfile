FROM golang:1.24

COPY . .

RUN go build -o subs bin/main.go

EXPOSE 8080

CMD ["./subs"]