FROM golang:1.20

WORKDIR /code

RUN apt update && apt install --yes --no-install-recommends protobuf-compiler
RUN go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
RUN go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
ENV PATH="$PATH:$GOROOT/bin"

COPY . .
RUN protoc --go_out=proto --go-grpc_out=proto proto/*.proto
RUN go mod download && go mod verify
RUN go build -o /app mafia_server/main.go

EXPOSE 9000

CMD /app
