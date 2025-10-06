//go:build file2llm_feature_ocr_tesseract || test

package ocr

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"unicode"

	"github.com/opengs/file2llm/ocr/gosseract"
	"github.com/opengs/file2llm/parser/bgra"
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

func (p *Tesseract) filterVisible(s string) string {
	var b strings.Builder
	b.Grow(len(s)) // preallocate memory

	for _, r := range s {
		if r == '\n' || (unicode.IsPrint(r) && (r == ' ' || unicode.IsLetter(r) || unicode.IsNumber(r) ||
			unicode.IsPunct(r) || unicode.IsSymbol(r))) {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func (p *Tesseract) OCR(ctx context.Context, image io.Reader) (string, error) {
	imageBytes, err := io.ReadAll(image)
	if err != nil {
		return "", errors.Join(errors.New("failed to read image bytes"), err)
	}

	p.lock.Lock()
	defer p.lock.Unlock()

	dpi, ok := ctx.Value("file2llm_DPI").(int)
	if !ok {
		dpi = 0
	}
	if err := p.client.SetVariable("user_defined_dpi", fmt.Sprintf("%d", dpi)); err != nil {
		return "", errors.Join(errors.New("failed to set DPI"), err)
	}

	if bytes.HasPrefix(imageBytes, bgra.RAWBGRA_HEADER) {
		img, err := bgra.ReadRAWBGRAImageFromBytes(imageBytes)
		if err != nil {
			return "", errors.Join(errors.New("failed to read bgra raw image data"), err)
		}
		rgbaImage := img.ConvertBGRAtoRGBAInplace()
		if err := p.client.SetImageFromRGBAImage(rgbaImage); err != nil {
			return "", errors.Join(errors.New("failed to prepare image for OCR"), err)
		}
	} else {
		if err := p.client.SetImageFromBytes(imageBytes); err != nil {
			return "", errors.Join(errors.New("failed to prepare image for OCR"), err)
		}
	}
	result, err := p.client.Text()
	if err != nil {
		return "", errors.Join(errors.New("OCR process failed"), err)
	}

	return p.filterVisible(result), nil
}

func (p *Tesseract) OCRWithProgress(ctx context.Context, image io.Reader) OCRProgress {
	progress := &tesseractOCRProgress{
		progressCh: make(chan uint8, 1),
	}
	progress.resultWaiter.Add(1)

	go func() {
		defer progress.resultWaiter.Done()
		defer close(progress.progressCh)

		imageBytes, err := io.ReadAll(image)
		if err != nil {
			progress.resultError = errors.Join(errors.New("failed to read image bytes"), err)
			return
		}

		p.lock.Lock()
		defer p.lock.Unlock()

		dpi, ok := ctx.Value("file2llm_DPI").(int)
		if !ok {
			dpi = 0
		}
		if err := p.client.SetVariable("user_defined_dpi", fmt.Sprintf("%d", dpi)); err != nil {
			progress.resultError = errors.Join(errors.New("failed to set DPI"), err)
			return
		}

		if bytes.HasPrefix(imageBytes, bgra.RAWBGRA_HEADER) {
			img, err := bgra.ReadRAWBGRAImageFromBytes(imageBytes)
			if err != nil {
				progress.resultError = errors.Join(errors.New("failed to read bgra raw image data"), err)
				return
			}
			rgbaImage := img.ConvertBGRAtoRGBAInplace()
			if err := p.client.SetImageFromRGBAImage(rgbaImage); err != nil {
				progress.resultError = errors.Join(errors.New("failed to prepare image for OCR"), err)
				return
			}

		} else {
			if err := p.client.SetImageFromBytes(imageBytes); err != nil {
				progress.resultError = errors.Join(errors.New("failed to prepare image for OCR"), err)
				return
			}
		}

		job, err := p.client.Recognize()
		if err != nil {
			progress.resultError = errors.Join(errors.New("failed to start OCR regognition"), err)
			return
		}

		for newCompletion := range job.Progress {
			progress.lastCompletion.Store(uint32(newCompletion))
			select {
			case progress.progressCh <- newCompletion:
			default:
			}
		}

		progress.resultError = job.Error
		progress.resultText = p.filterVisible(job.Result)
	}()

	return progress
}

func (p *Tesseract) Init() error {
	p.client = gosseract.NewClient()
	if err := p.client.SetLanguage(p.config.Languages...); err != nil {
		p.client.Close()
		return errors.Join(errors.New("failed to set languages"), err)
	}
	if err := p.client.SetVariable("tessedit_pageseg_mode", "1"); err != nil { // Automatic detection of image rotation. Build in function for set segmentation doesnt work, maybe bug in library
		p.client.Close()
		return errors.Join(errors.New("failed to set pageseg mode"), err)
	}
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
		if err := p.client.SetTessdataPrefix(p.getModelsFolder()); err != nil {
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

	if !slices.Contains(p.config.Languages, "osd") { // Orientation and script detection (OSD) model also must be loaded
		if _, err := os.Stat(p.getModelPath("osd")); err != nil {
			if errors.Is(err, os.ErrNotExist) {
				if downloadErr := p.downloadModel("osd"); downloadErr != nil {
					return errors.Join(errors.New("failed to download OSD model"), downloadErr)
				}
			} else {
				return errors.Join(errors.New("unexpected error while checking if OSD model exists"), err)
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
	return slices.Contains(p.config.SupportedImageFormats, mimeType) || mimeType == "image/file2llm-raw-bgra"
}

type tesseractOCRProgress struct {
	progressCh     chan uint8
	lastCompletion atomic.Uint32
	resultError    error
	resultText     string
	resultWaiter   sync.WaitGroup
}

func (t *tesseractOCRProgress) Completion() uint8 {
	return uint8(t.lastCompletion.Load())
}

func (t *tesseractOCRProgress) CompletionUpdates() chan uint8 {
	return t.progressCh
}

func (t *tesseractOCRProgress) Text() (string, error) {
	t.resultWaiter.Wait()
	return t.resultText, t.resultError
}
