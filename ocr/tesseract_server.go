package ocr

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"sync"
)

type TesseractServerConfig struct {
	// HTTP client used to make requests to the server
	Client *http.Client
	// Server base URL. For example http://127.0.0.1:8884
	BaseURL string
	// List of language codes that should be recognized. More languages - more processing time.
	// Order matters. Primary language has to go first as it will act as fallback. By default it will be ["eng"]
	// Make sure languages are installed on the server because default OCR server has only several languages enabled by default.
	Languages []string `json:"languages"`
}

func DefaultTesseractServerConfig() TesseractServerConfig {
	return TesseractServerConfig{
		Languages: []string{"eng"},
		BaseURL:   "http://127.0.0.1:8884",
		Client:    http.DefaultClient,
	}
}

// Uses tesseract server as OCR backend. https://github.com/otiai10/ocrserver
type TesseractServer struct {
	config TesseractServerConfig
}

func NewTesseractServer(config TesseractServerConfig) *TesseractServer {
	return &TesseractServer{
		config: config,
	}
}

func (p *TesseractServer) OCR(ctx context.Context, image io.Reader) (string, error) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	imagePart, err := writer.CreateFormFile("file", "data")
	if err != nil {
		return "", errors.Join(errors.New("failed to prepare multipart form data: failed to prepare image for sending as file"), err)
	}
	_, err = io.Copy(imagePart, image)
	if err != nil {
		return "", errors.Join(errors.New("failed to prepare multipart form data: failed to write image to multipart"), err)
	}

	var ocrOptions struct {
		Languages []string `json:"languages"`
	}
	ocrOptions.Languages = p.config.Languages
	ocrOptionsBytes, err := json.Marshal(ocrOptions)
	if err != nil {
		return "", errors.Join(errors.New("failed to marshall OCR options"), err)
	}

	if err = writer.WriteField("options", string(ocrOptionsBytes)); err != nil {
		return "", errors.Join(errors.New("failed to prepare multipart form data: failed to write options to multipart"), err)
	}

	err = writer.Close()
	if err != nil {
		return "", errors.Join(errors.New("failed to prepare multipart form data: failed to finalize writer"), err)
	}

	req, err := http.NewRequest("POST", p.config.BaseURL+"/tesseract", body)
	if err != nil {
		return "", errors.Join(errors.New("failed to prepare HTTP request"), err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := p.config.Client.Do(req)
	if err != nil {
		return "", errors.Join(errors.New("HTTP request to external server failed"), err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("bad status code from external sever: status code %d", resp.StatusCode)
	}

	var responseData struct {
		Data struct {
			Exit struct {
				Code uint `json:"code"`
			} `json:"exit"`
			StdErr string `json:"stderr"`
			StdOut string `json:"stdout"`
		} `json:"data"`
	}

	responseBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", errors.Join(errors.New("error while reading response body from remote server"), err)
	}

	if err := json.Unmarshal(responseBytes, &responseData); err != nil {
		return "", errors.Join(errors.New("failed to unmarshall response from remote server"), err)
	}

	if responseData.Data.Exit.Code != 0 {
		return "", fmt.Errorf("bad OCR execution status code: status code %d", responseData.Data.Exit.Code)
	}

	return responseData.Data.StdOut, nil
}

func (p *TesseractServer) OCRWithProgress(ctx context.Context, image io.Reader) OCRProgress {
	progress := &tesseractServerOCRProgress{
		progressCh: make(chan uint8, 1),
	}
	progress.resultWaiter.Add(1)

	go func() {
		defer progress.resultWaiter.Done()
		defer close(progress.progressCh)
		progress.resultText, progress.resultError = p.OCR(ctx, image)
	}()

	return progress
}

func (p *TesseractServer) IsMimeTypeSupported(mimeType string) bool {
	return mimeType == "image/jpeg" || mimeType == "image/png"
}

type tesseractServerOCRProgress struct {
	progressCh   chan uint8
	resultError  error
	resultText   string
	resultWaiter sync.WaitGroup
}

func (t *tesseractServerOCRProgress) Completion() uint8 {
	return 0
}

func (t *tesseractServerOCRProgress) CompletionUpdates() chan uint8 {
	return t.progressCh
}

func (t *tesseractServerOCRProgress) Text() (string, error) {
	t.resultWaiter.Wait()
	return t.resultText, t.resultError
}
