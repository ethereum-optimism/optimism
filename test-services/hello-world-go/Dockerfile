FROM golang:1.21.0

WORKDIR /app

COPY ./ ./

RUN CGO_ENABLED=0 GOOS=linux go build -o /main

EXPOSE 2001

CMD [ "./start.sh" ]
