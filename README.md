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

## Features

To make things easier, all the features that requires additional libraries to be installed or requre CGO have theirs build flags.

| -- | -- | -- | -- |
| Type | Requires CGO | Required tags | Required OCR |
| -- | -- | -- | -- |
| image/png | NO |  | YES |
| image/jpeg | NO |  | YES |
| image/webp | NO |  | YES |


| -- | -- | -- |
| OCR Provider | Required CGO | Required tags |
| -- | -- | -- |
| Tesseract OCR | YES | file2llm_feature_tesseract |
| Pabble OCR | NO |  |
| MMOCR | NO | |

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

