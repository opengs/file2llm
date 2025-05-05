package ocr

import (
	"context"
	"errors"
	"slices"
	"sync"

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

func (p *TesseractPool) OCR(ctx context.Context, image []byte) (string, error) {
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

func (p *TesseractPool) IsMimeTypeSupported(mimeType string) bool {
	return slices.Contains(p.workerConfig.SupportedImageFormats, mimeType)
}
