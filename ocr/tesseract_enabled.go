//go:build file2llm_feature_ocr_tesseract || test

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
	"slices"
	"sync"

	"github.com/otiai10/gosseract/v2"
)

const FeatureTesseractEnabled = true

type Tesseract struct {
	client *gosseract.Client
	lock   sync.Mutex
	config TesseractConfig
}

func NewTesseract(config TesseractConfig) *Tesseract {
	return &Tesseract{
		config: config,
	}
}

func (p *Tesseract) OCR(ctx context.Context, image []byte) (string, error) {
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

func (p *Tesseract) Init() error {
	p.client = gosseract.NewClient()
	p.client.SetLanguage(p.config.Languages...)
	if err := p.client.DisableOutput(); err != nil {
		p.client.Close()
		return errors.Join(errors.New("failed to disable logs"), err)
	}
	for key, val := range p.config.Variables {
		if err := p.client.SetVariable(gosseract.SettableVariable(key), val); err != nil {
			p.client.Close()
			return errors.Join(fmt.Errorf("failed to set variable [%s]", key), err)
		}
	}
	if p.config.LoadCustomModels {
		if err := p.loadModels(); err != nil {
			p.client.Close()
			return errors.Join(errors.New("failed to load language models"), err)
		}
		if err := p.client.SetTessdataPrefix(p.config.ModelsFolder); err != nil {
			p.client.Close()
			return errors.Join(errors.New("failed to set custom models folder"), err)
		}
	}
	return nil
}
func (p *Tesseract) Destroy() error {
	return p.client.Close()
}

func (p *Tesseract) getModelDownloadLink(language string) string {
	var ocrModelLinkByType map[TesseractModelType]string = map[TesseractModelType]string{
		TesseractModelFast:        "https://github.com/tesseract-ocr/tessdata_fast/raw/refs/heads/main/",
		TesseractModelNormal:      "https://github.com/tesseract-ocr/tessdata/raw/refs/heads/main/",
		TesseractModelBestQuality: "https://github.com/tesseract-ocr/tessdata_best/raw/refs/heads/main/",
	}
	return ocrModelLinkByType[p.config.ModelType] + language + ".traineddata"
}

func (p *Tesseract) getModelsFolder() string {
	return path.Join(p.config.ModelsFolder, string(p.config.ModelType))
}

func (p *Tesseract) getModelPath(language string) string {
	return path.Join(p.getModelsFolder(), language+".traineddata")
}

func (p *Tesseract) loadModels() error {
	if err := os.MkdirAll(p.getModelsFolder(), 0700); err != nil {
		return errors.Join(errors.New("failed to create folder for models"), err)
	}

	for _, language := range p.config.Languages {
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

func (p *Tesseract) downloadModel(language string) error {
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

func (p *Tesseract) IsMimeTypeSupported(mimeType string) bool {
	return slices.Contains(p.config.SupportedImageFormats, mimeType)
}
