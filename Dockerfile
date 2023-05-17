FROM golang
RUN mkdir /app
COPY ./pkg/ /app/pkg
COPY . /app
WORKDIR /app
RUN go mod download
RUN go mod verify
RUN go test ./... -cover
RUN go build -o /app/main /app/main.go
ENTRYPOINT ["/app/main"]