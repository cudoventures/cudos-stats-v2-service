FROM amd64/golang:1.17-buster

RUN apt-get update && apt-get install git

WORKDIR /usr/src/cudos-stats-v2-service

COPY . .

EXPOSE 3000

RUN go build -mod=readonly ./cmd/stats-service

CMD ["/bin/bash", "-c", "./stats-service"]