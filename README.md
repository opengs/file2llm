<h1 align="center">
  File to LLM
</h1>
<h4 align="center">GO library to convert files of multiple formats to text understandable by LLM</h4>

<p align="center">
  <img alt="application/pdf" src="https://img.shields.io/badge/PDF-lightgray?style=for-the-badge">
  <img alt="application/msword" src="https://img.shields.io/badge/DOC-gray?style=for-the-badge">
  <img alt="application/vnd.openxmlformats-officedocument.wordprocessingml.document" src="https://img.shields.io/badge/DOCX-gray?style=for-the-badge">
  <img alt="application/vnd.ms-powerpoint" src="https://img.shields.io/badge/PPT-gray?style=for-the-badge">
  <img alt="application/application/vnd.openxmlformats-officedocument.presentationml.presentation" src="https://img.shields.io/badge/PPTX-gray?style=for-the-badge">
  <img alt="application/vnd.oasis.opendocument.text" src="https://img.shields.io/badge/ODT-gray?style=for-the-badge">
  <img alt="application/vnd.apple.pages" src="https://img.shields.io/badge/PAGES-gray?style=for-the-badge">
  <img alt="application/rtf" src="https://img.shields.io/badge/RTF-gray?style=for-the-badge">
  <br>
  <img alt="image/png" src="https://img.shields.io/badge/PNG-lightgray?style=for-the-badge">
  <img alt="image/jpeg" src="https://img.shields.io/badge/JPEG-lightgray?style=for-the-badge">
  <img alt="image/webp" src="https://img.shields.io/badge/WEBP-lightgray?style=for-the-badge">
  <img alt="image/bmp" src="https://img.shields.io/badge/BMP-lightgray?style=for-the-badge">
  <img alt="image/gif" src="https://img.shields.io/badge/GIF-lightgray?style=for-the-badge">
  <img alt="image/tiff" src="https://img.shields.io/badge/TIFF-lightgray?style=for-the-badge">
  <br>
  <img alt="application/zip" src="https://img.shields.io/badge/ZIP-gray?style=for-the-badge">
  <img alt="application/vnd.rar" src="https://img.shields.io/badge/RAR-gray?style=for-the-badge">
  <img alt="application/x-7z-compressed" src="https://img.shields.io/badge/7Z-gray?style=for-the-badge">
  <img alt="application/gzip" src="https://img.shields.io/badge/GZ-gray?style=for-the-badge">
  <img alt="application/tar" src="https://img.shields.io/badge/TAR-gray?style=for-the-badge">
  <img alt="application/x-bzip2" src="https://img.shields.io/badge/BZ2-gray?style=for-the-badge">
</p>

File2LLM is specifically designed to work with LLMs. Unlike other Golang solutions, it preserves text location, padding, and formatting, adding structural boundaries that are understandable by LLMs. It also performs additional processing to ensure that the extracted text is properly interpretable by LLMs.

File2LLM can handle nested file formats (such as archives) by recursively reading them and creating structured file information suitable for LLM input.

## Example

Get the main `file2llm` library

```bash
go get -u github.com/opengs/file2llm
```

Install dependencies to work with PDF and images (OCR). This is optional.

```bash
sudo apt install -y libpoppler-glib-dev libcairo2 libcairo2-dev libtesseract-dev
```

This will extract text from PDF including images

```go
package main

import (
	"context"
	"os"

	"github.com/opengs/file2llm/ocr"
	"github.com/opengs/file2llm/parser"
)

func main() {
	fp, err := os.Open("file.pdf")
	if err != nil {
		panic(err.Error())
	}
	defer fp.Close()

  // Initialize OCR to be able to extract text from images
	ocrProvider := ocr.NewTesseractProvider(ocr.DefaultTesseractConfig())
	if err := ocrProvider.Init(); err != nil {
		panic(err.Error())
	}
	defer ocrProvider.Destroy()

	p := parser.New(ocrProvider)
	result := p.Parse(context.Background(), fp)
	println(result.String())
}
```

Run code with build tags to enable features from `file2llm` library.

```bash
go run -tags=file2llm_feature_tesseract,file2llm_feature_pdf main.go
```

## Features

|      | CGO | Build tags           | Requires OCR | Required libraries                                          | Notes                                                    |
| ---- | --- | -------------------- | ------------ | ----------------------------------------------------------- | -------------------------------------------------------- |
| png  | NO  |                      | YES          |                                                             |                                                          |
| jpeg | NO  |                      | YES          |                                                             |                                                          |
| webp | NO  |                      | YES          |                                                             |                                                          |
| gif  | NO  |                      | YES          |                                                             | Extracts first frame                                     |
| bmp  | NO  |                      | YES          |                                                             |                                                          |
| tiff | NO  |                      | YES          |                                                             |                                                          |
| pdf  | YES | file2llm_feature_pdf | optional     | libpoppler-glib libpoppler-glib-dev libcairo2 libcairo2-dev | Extracts text from embeded images using OCR if available |

| OCR Provider     | CGO | Required tags              | Required libraries         |
| ---------------- | --- | -------------------------- | -------------------------- |
| Tesseract        | YES | file2llm_feature_tesseract | tesseract libtesseract-dev |
| Tesseract Server | NO  |                            |                            |
| Pabble OCR       | NO  |                            |                            |
| MMOCR            | NO  |                            |                            |

## Standalone usage with Docker

```bash
docker run \
    -p 8080:8080
    -v ./data/file2llm:/data
    ghc
```

Docker image is precompiled with all the features enabled.

## License
file2llm is distributed under AGPL3.0 license. If you need close code commercial use

