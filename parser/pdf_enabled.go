//go:build file2llm_feature_pdf || test

package parser

/*
 #cgo pkg-config: poppler-glib cairo
 #include <poppler.h>
 #include <cairo.h>
 #include <cairo-pdf.h>
 #include <stdlib.h>
*/
import "C"
import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"unsafe"

	"github.com/opengs/file2llm/parser/bgra"
)

const FeaturePDFEnabled = true

// Parses `application/pdf` files
type PDFParser struct {
	innerParser Parser

	dpi uint32
}

func NewPDFParser(innerParser Parser, dpi uint32) *PDFParser {
	return &PDFParser{
		innerParser: innerParser,

		dpi: dpi, // Ideal for ocr
	}
}

func (p *PDFParser) SupportedMimeTypes() []string {
	return []string{"application/pdf"}
}

func (p *PDFParser) Parse(ctx context.Context, file io.Reader, path string) Result {
	pdfData, err := io.ReadAll(file)
	if err != nil {
		return &PDFParserResult{Err: errors.Join(errors.New("failed to read data to the bytes buffer"), err)}
	}

	gbytes := C.g_bytes_new(C.gconstpointer(unsafe.Pointer(&pdfData[0])), C.size_t(len(pdfData)))
	if gbytes == nil {
		return &PDFParserResult{Err: errors.New("failed to create GBytes")}
	}
	defer C.g_bytes_unref(gbytes)

	var gErr *C.GError
	doc := C.poppler_document_new_from_bytes(gbytes, nil, &gErr)
	if doc == nil {
		if gErr != nil {
			defer C.g_error_free(gErr)
			return &PDFParserResult{
				Err: errors.Join(ErrBadFile, errors.New(C.GoString((*C.char)(gErr.message)))),
			}
		}
		return &PDFParserResult{
			Err: errors.New("unknown error while reading PDF document"),
		}
	}
	defer C.g_object_unref(C.gpointer(doc))

	// Get the metadata
	var meta string
	metaCStr := C.poppler_document_get_metadata(doc)
	if metaCStr != nil {
		meta = C.GoString(metaCStr)
		C.g_free(C.gpointer(metaCStr))
	}

	var pages []string

	n_pages := int(C.poppler_document_get_n_pages(doc))
	for pageIndex := range n_pages {
		page := C.poppler_document_get_page(doc, C.int(pageIndex))
		if page == nil {
			return &PDFParserResult{
				FullPath: path,
				Metadata: meta,
				Err:      errors.Join(ErrBadFile, fmt.Errorf("failed to get page %d from the document", pageIndex)),
			}
		}

		pageImage, err := p.getPageImage(page)
		C.g_object_unref(C.gpointer(page))
		if err != nil {
			return &PDFParserResult{
				FullPath: path,
				Metadata: meta,
				Err:      errors.Join(fmt.Errorf("failed to render page %d", pageIndex), err),
			}
		}

		imageResult := p.innerParser.Parse(context.WithValue(ctx, "file2llm_DPI", p.dpi), pageImage, "")
		if imageResult.Error() != nil {
			return &PDFParserResult{
				FullPath: path,
				Metadata: meta,
				Err:      errors.Join(fmt.Errorf("failed to parse page %d", pageIndex), imageResult.Error()),
			}
		}
		pages = append(pages, imageResult.String())
	}

	return &PDFParserResult{Pages: pages, Metadata: meta}
}

func (p *PDFParser) ParseStream(ctx context.Context, file io.Reader, path string) StreamResultIterator {
	return &PDFStreamResultIterator{
		pdfParser: p,
		ctx:       ctx,
		file:      file,
		path:      path,
	}
}

func (p *PDFParser) getPageImage(page *C.PopplerPage) (io.Reader, error) {
	scale := float64(p.dpi) / 72.0

	var w, h C.double
	C.poppler_page_get_size(page, &w, &h)
	width := int(float64(w) * scale)
	if width <= 0 {
		return nil, fmt.Errorf("page width is: %d", width)
	}
	height := int(float64(h) * scale)
	if height <= 0 {
		return nil, fmt.Errorf("page height is: %d", height)
	}

	surface := C.cairo_image_surface_create(C.CAIRO_FORMAT_ARGB32, C.int(int(width)), C.int(int(height)))
	if surface == nil {
		return nil, errors.New("failed to create cairo surface")
	}

	cr := C.cairo_create(surface)
	if cr == nil {
		C.cairo_surface_destroy(surface)
		return nil, errors.New("failed to create cairo context")
	}

	C.cairo_scale(cr, C.double(scale), C.double(scale))
	C.cairo_set_source_rgb(cr, 1.0, 1.0, 1.0)
	C.cairo_paint(cr)

	C.poppler_page_render(page, cr)
	imgWidth := int(C.cairo_image_surface_get_width(surface))
	imgHeight := int(C.cairo_image_surface_get_height(surface))
	imgStride := int(C.cairo_image_surface_get_stride(surface))
	imgData := C.cairo_image_surface_get_data(surface)
	if imgData == nil {
		C.cairo_destroy(cr)
		C.cairo_surface_destroy(surface)
		return nil, errors.New("failed to get rendered image data")
	}
	bufSize := imgHeight * imgStride
	buf := C.GoBytes(unsafe.Pointer(imgData), C.int(bufSize))
	C.cairo_destroy(cr)
	C.cairo_surface_destroy(surface)

	sizes := make([]byte, 8+8+8)
	binary.BigEndian.PutUint64(sizes, uint64(imgWidth))
	binary.BigEndian.PutUint64(sizes[8:], uint64(imgHeight))
	binary.BigEndian.PutUint64(sizes[16:], uint64(imgStride))
	bgraStream := io.MultiReader(
		bytes.NewBuffer(bgra.RAWBGRA_HEADER),
		bytes.NewBuffer(sizes),
		bytes.NewBuffer(buf),
	)

	return bgraStream, nil
}

type PDFStreamResultIterator struct {
	pdfParser *PDFParser
	ctx       context.Context
	file      io.Reader
	path      string
	started   bool
	completed bool

	docBuffer        *C.GBytes
	doc              *C.PopplerDocument
	nPages           int
	currentPage      *C.PopplerPage
	currentPageIndex int

	pageProcessing StreamResultIterator
	current        StreamResult
}

func (i *PDFStreamResultIterator) Current() StreamResult {
	return i.current
}

func (i *PDFStreamResultIterator) Next(ctx context.Context) bool {
	if i.completed {
		i.current = nil
		return false
	}

	if !i.started {
		i.started = true
		i.current = &PDFParserStreamResult{
			FullPath:     i.path,
			CurrentStage: ProgressNew,
		}
		return true
	}

	if i.doc == nil {
		pdfData, err := io.ReadAll(i.file)
		if err != nil {
			i.completed = true
			i.current = &PDFParserStreamResult{
				FullPath:     i.path,
				Err:          errors.Join(errors.New("failed to read data to the bytes buffer"), err),
				CurrentStage: ProgressCompleted,
			}
			return true
		}

		i.docBuffer = C.g_bytes_new(C.gconstpointer(unsafe.Pointer(&pdfData[0])), C.size_t(len(pdfData)))
		if i.docBuffer == nil {
			i.completed = true
			i.current = &PDFParserStreamResult{
				FullPath:     i.path,
				Err:          errors.New("failed to create GBytes"),
				CurrentStage: ProgressCompleted,
			}
			return true
		}

		var gErr *C.GError
		i.doc = C.poppler_document_new_from_bytes(i.docBuffer, nil, &gErr)
		if i.doc == nil {
			C.g_bytes_unref(i.docBuffer)
			i.docBuffer = nil
			if gErr != nil {
				defer C.g_error_free(gErr)
				i.current = &PDFParserStreamResult{
					FullPath:     i.path,
					Err:          errors.Join(ErrBadFile, errors.New(C.GoString((*C.char)(gErr.message)))),
					CurrentStage: ProgressCompleted,
				}
				return true
			}
			i.current = &PDFParserStreamResult{
				FullPath:     i.path,
				Err:          errors.New("unknown error while reading PDF document"),
				CurrentStage: ProgressCompleted,
			}
			return true
		}

		var meta string
		metaCStr := C.poppler_document_get_metadata(i.doc)
		if metaCStr != nil {
			meta = C.GoString(metaCStr)
			C.g_free(C.gpointer(metaCStr))
		}
		i.nPages = int(C.poppler_document_get_n_pages(i.doc))

		i.current = &PDFParserStreamResult{
			FullPath:     i.path,
			CurrentStage: ProgressUpdate,
			Text:         fmt.Sprintf("------METADATA START------\n\n%s------METADATA END------\n\n", meta),
		}
		return true
	}

	if i.pageProcessing != nil {
		if i.pageProcessing.Next(ctx) {
			nextPageUpdate := i.pageProcessing.Current()
			if err := nextPageUpdate.Error(); err != nil {
				i.completed = true
				i.current = &PDFParserStreamResult{
					FullPath:     i.path,
					CurrentStage: ProgressCompleted,
					Err:          errors.Join(fmt.Errorf("failed to process page %d of the file", i.currentPageIndex-1), err),
				}
				return true
			}

			if nextPageUpdate.Stage() == ProgressNew || nextPageUpdate.Stage() == ProgressUpdate {
				progressPerPage := 1.0 / float64(i.nPages)
				progressInsidePage := float64(nextPageUpdate.Progress()) / 100
				curentProgress := progressPerPage * (float64(i.currentPageIndex-1) + progressInsidePage) * 100

				i.current = &PDFParserStreamResult{
					FullPath:        i.path,
					CurrentStage:    ProgressUpdate,
					CurrentProgress: uint8(curentProgress),
					Text:            nextPageUpdate.String(),
				}
				return true
			}

			if nextPageUpdate.Stage() == ProgressCompleted {
				text := nextPageUpdate.String()
				i.pageProcessing.Close()
				i.pageProcessing = nil
				i.current = &PDFParserStreamResult{
					FullPath:        i.path,
					CurrentStage:    ProgressUpdate,
					CurrentProgress: uint8(1.0 / float64(i.nPages) * float64(i.currentPageIndex) * 100),
					Text:            text,
				}
				return true
			}
		} else {
			i.pageProcessing.Close()
			i.pageProcessing = nil
		}
	}

	if i.currentPageIndex < i.nPages {
		if i.currentPage != nil {
			C.g_object_unref(C.gpointer(i.currentPage))
		}

		i.currentPage = C.poppler_document_get_page(i.doc, C.int(i.currentPageIndex))
		if i.currentPage == nil {
			i.completed = true
			i.current = &PDFParserStreamResult{
				FullPath:     i.path,
				CurrentStage: ProgressCompleted,
				Err:          fmt.Errorf("failed to load page %d of the file", i.currentPageIndex),
			}
			return true
		}
		i.currentPageIndex += 1

		imageData, err := i.pdfParser.getPageImage(i.currentPage)
		if err != nil {
			i.completed = true
			i.current = &PDFParserStreamResult{
				FullPath:     i.path,
				CurrentStage: ProgressCompleted,
				Err:          fmt.Errorf("failed to render page %d of the file", i.currentPageIndex-1),
			}
			return true
		}
		i.pageProcessing = i.pdfParser.innerParser.ParseStream(context.WithValue(i.ctx, "file2llm_DPI", i.pdfParser.dpi), imageData, "")

		return i.Next(ctx)
	}

	if !i.completed {
		i.completed = true
		i.current = &PDFParserStreamResult{
			FullPath:     i.path,
			CurrentStage: ProgressCompleted,
		}
		return true
	}

	i.current = nil
	return false
}

func (i *PDFStreamResultIterator) Close() {
	if i.pageProcessing != nil {
		i.pageProcessing.Close()
	}

	if i.currentPage != nil {
		C.g_object_unref(C.gpointer(i.currentPage))
	}

	if i.docBuffer != nil {
		C.g_bytes_unref(i.docBuffer)
		C.g_object_unref(C.gpointer(i.doc))
	}
}
