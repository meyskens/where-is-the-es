FROM node:22 AS frontend
WORKDIR /app
COPY ./where-is-the-es /app/
RUN npm install
RUN npm run build

FROM golang:1.24-alpine AS backend

COPY ./ /go/src/github.com/meyskens/where-is-the-es/
RUN rm /go/src/github.com/meyskens/where-is-the-es/cmd/wites/frontend/*
COPY --from=frontend /app/build/client /go/src/github.com/meyskens/where-is-the-es/cmd/wites/frontend
WORKDIR /go/src/github.com/meyskens/where-is-the-es/

RUN go mod download
RUN go build ./cmd/wites

FROM alpine:latest
COPY --from=backend /go/src/github.com/meyskens/where-is-the-es/wites /usr/local/bin/

COPY --from=frontend /app/build/client /go/src/github.com/meyskens/where-is-the-es/cmd/wites/frontend


EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/wites"]
CMD ["serve"]