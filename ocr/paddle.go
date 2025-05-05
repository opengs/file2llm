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
	"strings"
)

type PaddleConfig struct {
	// HTTP client used to make requests to the server
	Client *http.Client
	// Server base URL. For example http://127.0.0.1:8884
	BaseURL string
	// List of language codes that should be recognized. More languages - more processing time.
	// Order matters. Primary language has to go first as it will act as fallback. By default it will be ["eng"]
	// Make sure languages are installed on the server because default OCR server has only several languages enabled by default.
	Languages []string `json:"languages"`
}

func DefaultPaddleConfig() PaddleConfig {
	return PaddleConfig{
		Languages: []string{"eng"},
		BaseURL:   "http://127.0.0.1:8884",
		Client:    http.DefaultClient,
	}
}

type Paddle struct {
	config PaddleConfig
}

func NewPaddle(config PaddleConfig) *Paddle {
	return &Paddle{
		config: config,
	}
}

func (p *Paddle) OCR(ctx context.Context, image []byte) (string, error) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	imagePart, err := writer.CreateFormFile("file", "data")
	if err != nil {
		return "", errors.Join(errors.New("failed to prepare multipart form data: failed to prepare image for sending as file"), err)
	}
	_, err = io.Copy(imagePart, bytes.NewReader(image))
	if err != nil {
		return "", errors.Join(errors.New("failed to prepare multipart form data: failed to write image to multipart"), err)
	}

	if err = writer.WriteField("languages", strings.Join(p.config.Languages, ",")); err != nil {
		return "", errors.Join(errors.New("failed to prepare multipart form data: failed to write languages to multipart"), err)
	}

	err = writer.Close()
	if err != nil {
		return "", errors.Join(errors.New("failed to prepare multipart form data: failed to finalize writer"), err)
	}

	req, err := http.NewRequest("POST", p.config.BaseURL+"/ocr", body)
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
		Text string `json:"text"`
	}

	responseBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", errors.Join(errors.New("error while reading response body from remote server"), err)
	}

	if err := json.Unmarshal(responseBytes, &responseData); err != nil {
		return "", errors.Join(errors.New("failed to unmarshall response from remote server"), err)
	}

	return responseData.Text, nil
}

func (p *Paddle) IsMimeTypeSupported(mimeType string) bool {
	return mimeType == "image/jpeg" || mimeType == "image/png"
}
