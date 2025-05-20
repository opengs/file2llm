package ocr

import (
	"context"
	"errors"
	"io"
	"slices"
	"sync"
	"sync/atomic"

	"golang.org/x/sync/semaphore"
)

type TesseractPool struct {
	size         uint32
	workerConfig TesseractConfig
	workers      []*Tesseract

	workLock             *semaphore.Weighted
	poolManipulationLock sync.Mutex
}

func NewTesseractPool(size uint32, workerConfig TesseractConfig) *TesseractPool {
	return &TesseractPool{
		size:         size,
		workerConfig: workerConfig,
		workers:      make([]*Tesseract, 0, size),
		workLock:     semaphore.NewWeighted(int64(size)),
	}
}

func (p *TesseractPool) Init(ctx context.Context) error {
	if err := p.workLock.Acquire(ctx, int64(p.size)); err != nil {
		return errors.Join(errors.New("failed to accuire exclusive lock on entire pool"), err)
	}
	defer p.workLock.Release(int64(p.size))

	for range p.size {
		worker := NewTesseract(p.workerConfig)
		if err := worker.Init(); err != nil {
			var allErrors = []error{err}
			for _, w := range p.workers {
				if err := w.Destroy(); err != nil {
					allErrors = append(allErrors, err)
				}
			}

			return errors.Join(allErrors...)
		}
		p.workers = append(p.workers, worker)
	}
	return nil
}

func (p *TesseractPool) Destroy(ctx context.Context) error {
	if err := p.workLock.Acquire(ctx, int64(p.size)); err != nil {
		return errors.Join(errors.New("failed to accuire exclusive lock on entire pool"), err)
	}
	defer p.workLock.Release(int64(p.size))

	var destroyErrors []error
	for _, w := range p.workers {
		if err := w.Destroy(); err != nil {
			destroyErrors = append(destroyErrors, err)
		}
	}

	if len(destroyErrors) > 0 {
		return errors.Join(destroyErrors...)
	} else {
		return nil
	}
}

func (p *TesseractPool) OCR(ctx context.Context, image io.Reader) (string, error) {
	if err := p.workLock.Acquire(ctx, 1); err != nil {
		return "", errors.Join(errors.New("failed to accuire work lock"), err)
	}
	defer p.workLock.Release(1)

	p.poolManipulationLock.Lock()
	if len(p.workers) == 0 { // in case if it is not initialized
		p.poolManipulationLock.Unlock()
		return "", errors.New("pool is empty")
	}
	worker := p.workers[len(p.workers)-1]
	p.workers = p.workers[:len(p.workers)-1]
	p.poolManipulationLock.Unlock()

	resultString, resultErr := worker.OCR(ctx, image)

	p.poolManipulationLock.Lock()
	p.workers = append(p.workers, worker)
	p.poolManipulationLock.Unlock()

	return resultString, resultErr
}

func (p *TesseractPool) OCRWithProgress(ctx context.Context, image io.Reader) OCRProgress {
	pooledProcess := &tesseractPoolOCRProgress{
		progressCh: make(chan uint8, 1),
	}
	pooledProcess.resultWaiter.Add(1)

	go func() {
		defer pooledProcess.resultWaiter.Done()
		defer close(pooledProcess.progressCh)

		if err := p.workLock.Acquire(ctx, 1); err != nil {
			pooledProcess.resultError = errors.Join(errors.New("failed to accuire work lock"), err)
			return
		}
		defer p.workLock.Release(1)

		p.poolManipulationLock.Lock()
		if len(p.workers) == 0 { // in case if it is not initialized
			p.poolManipulationLock.Unlock()
			pooledProcess.resultError = errors.New("pool is empty")
			return
		}
		worker := p.workers[len(p.workers)-1]
		p.workers = p.workers[:len(p.workers)-1]
		p.poolManipulationLock.Unlock()

		progress := worker.OCRWithProgress(ctx, image)
		for p := range progress.CompletionUpdates() {
			pooledProcess.lastCompletion.Store(uint32(p))
			select {
			case pooledProcess.progressCh <- p:
			default:
			}
		}
		pooledProcess.resultText, pooledProcess.resultError = progress.Text()
		p.poolManipulationLock.Lock()
		p.workers = append(p.workers, worker)
		p.poolManipulationLock.Unlock()
	}()

	return pooledProcess
}

func (p *TesseractPool) IsMimeTypeSupported(mimeType string) bool {
	return slices.Contains(p.workerConfig.SupportedImageFormats, mimeType)
}

type tesseractPoolOCRProgress struct {
	progressCh     chan uint8
	lastCompletion atomic.Uint32
	resultError    error
	resultText     string
	resultWaiter   sync.WaitGroup
}

func (t *tesseractPoolOCRProgress) Completion() uint8 {
	return uint8(t.lastCompletion.Load())
}

func (t *tesseractPoolOCRProgress) CompletionUpdates() chan uint8 {
	return t.progressCh
}

func (t *tesseractPoolOCRProgress) Text() (string, error) {
	t.resultWaiter.Wait()
	return t.resultText, t.resultError
}
