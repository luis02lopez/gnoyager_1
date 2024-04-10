FROM golang:1.22
COPY . /app
WORKDIR /app
RUN go mod tidy
CMD [ "go", "run", "." ]
