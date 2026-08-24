FROM golang:1.23.12
ENV GOPROXY=off
ENV GOSUMDB=off
WORKDIR /src
COPY . .
RUN go build -mod=vendor ./...
CMD ["bash"]
