//go:build !file2llm_feature_ocr_tesseract

package ocr

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"sync"

	"github.com/otiai10/gosseract/v2"
)

const FeatureTesseractEnabled = true

type tesseractProvider struct {
	client       *gosseract.Client
	lock         sync.Mutex
	languages    []string
	modelType    TesseractModelType
	modelsFolder string
}

func NewTesseractProvider(languages []string, modelType TesseractModelType, modelsFolder string) Provider {
	return &tesseractProvider{
		languages:    languages,
		modelType:    modelType,
		modelsFolder: modelsFolder,
	}
}

func (p *tesseractProvider) OCR(ctx context.Context, image []byte) (string, error) {
	p.lock.Lock()
	defer p.lock.Unlock()

	if err := p.client.SetImageFromBytes(image); err != nil {
		return "", errors.Join(errors.New("failed to prepare image for OCR"), err)
	}
	result, err := p.client.Text()
	if err != nil {
		return "", errors.Join(errors.New("OCR process failed"), err)
	}

	return result, nil
}

func (p *tesseractProvider) Init() error {
	p.client = gosseract.NewClient()
	p.client.SetLanguage(p.languages...)
	p.client.SetVariable("load_system_dawg", "0")
	p.client.SetVariable("load_freq_dawg", "0")
	if err := p.loadModels(); err != nil {
		p.client.Close()
		return errors.Join(errors.New("failed to load language models"), err)
	}
	return nil
}
func (p *tesseractProvider) Destroy() error {
	return p.client.Close()
}

func (p *tesseractProvider) getModelDownloadLink(language string) string {
	var ocrModelLinkByType map[TesseractModelType]string = map[TesseractModelType]string{
		TesseractModelFast:        "https://github.com/tesseract-ocr/tessdata_fast/raw/refs/heads/main/",
		TesseractModelNormal:      "https://github.com/tesseract-ocr/tessdata/raw/refs/heads/main/",
		TesseractModelBestQuality: "https://github.com/tesseract-ocr/tessdata_best/raw/refs/heads/main/",
	}
	return ocrModelLinkByType[p.modelType] + language + ".traineddata"
}

func (p *tesseractProvider) getModelsFolder() string {
	return path.Join(p.modelsFolder, string(p.modelType))
}

func (p *tesseractProvider) getModelPath(language string) string {
	return path.Join(p.getModelsFolder(), language+".traineddata")
}

func (p *tesseractProvider) loadModels() error {
	if err := os.MkdirAll(p.getModelsFolder(), 0700); err != nil {
		return errors.Join(errors.New("failed to create folder for models"), err)
	}

	for _, language := range p.languages {
		if _, err := os.Stat(p.getModelPath(language)); err != nil {
			if errors.Is(err, os.ErrNotExist) {
				if downloadErr := p.downloadModel(language); downloadErr != nil {
					return errors.Join(errors.New("failed to download language model "+language), downloadErr)
				}
			} else {
				return errors.Join(errors.New("unexpected error while checking if model exists"), err)
			}
		}
	}

	return nil
}

func (p *tesseractProvider) downloadModel(language string) error {
	resp, err := http.Get(p.getModelDownloadLink(language))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bad status: %s", resp.Status)
	}

	tmpFile, err := os.CreateTemp(filepath.Dir(p.getModelPath(language)), "*.tmp")
	if err != nil {
		return err
	}
	defer func() {
		tmpFile.Close()
		os.Remove(tmpFile.Name())
	}()

	if _, err := io.Copy(tmpFile, resp.Body); err != nil {
		return err
	}

	if err := tmpFile.Close(); err != nil {
		return err
	}

	return os.Rename(tmpFile.Name(), p.getModelPath(language))
}

func (p *tesseractProvider) Name() ProviderName {
	return ProviderNameTesseract
}

func (p *tesseractProvider) IsMimeTypeSupported(mimeType string) bool {
	return mimeType == "image/jpeg" || mimeType == "image/png"
}
