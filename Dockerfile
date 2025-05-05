FROM --platform=${BUILDPLATFORM:-linux/amd64} golang:1.24.2-alpine AS builder
RUN apk add --no-cache git gcc g++ make build-base tesseract-ocr tesseract-ocr-dev cairo cairo-dev poppler poppler-dev poppler-glib
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=1 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH:-amd64} go build -tags=file2llm_feature_tesseract,file2llm_feature_pdf -ldflags="-w -s" -o /app/app cmd/main.go cmd/serve.go

FROM alpine:3.21.3
LABEL org.opencontainers.image.source="https://github.com/opengs/file2llm"
EXPOSE 8884
RUN apk add --no-cache tesseract-ocr libpng libjpeg zlib tiff libwebp giflib openjpeg-tools cairo poppler poppler-glib
COPY --from=builder /app/app /app
ENTRYPOINT ["/app"]
CMD ["serve"]