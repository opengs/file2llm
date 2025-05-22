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
	"math"
	pathlib "path"
	"strings"
	"unsafe"

	"github.com/opengs/file2llm/parser/bgra"
)

const FeaturePDFEnabled = true

// Parses `application/pdf` files
type PDFParser struct {
	innerParser Parser
}

func NewPDFParser(innerParser Parser) *PDFParser {
	return &PDFParser{
		innerParser: innerParser,
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

	var pages []PDFParserResultPage

	n_pages := int(C.poppler_document_get_n_pages(doc))
	for pageIndex := range n_pages {
		page := C.poppler_document_get_page(doc, C.int(pageIndex))
		if page == nil {
			continue
		}
		pageResult := PDFParserResultPage{}

		// Get all the text from the page
		textC := C.poppler_page_get_text(page)
		if unsafe.Pointer(textC) != nil {
			text := C.GoString(textC)
			pageResult.Text = text
			C.g_free(C.gpointer(textC))
		}

		// Get all the images on the page
		var pageImages []Result
		imageMappingList := C.poppler_page_get_image_mapping(page)
		if imageMappingList != nil {
			for l := imageMappingList; l != nil; l = l.next {
				mapping := (*C.PopplerImageMapping)(l.data)
				cImage := C.poppler_page_get_image(page, mapping.image_id)
				if cImage == nil {
					continue
				}

				format := C.cairo_image_surface_get_format(cImage)
				if format != C.CAIRO_FORMAT_ARGB32 && format != C.CAIRO_FORMAT_RGB24 {
					C.cairo_surface_destroy(cImage)
					continue
				}

				width := int(C.cairo_image_surface_get_width(cImage))
				height := int(C.cairo_image_surface_get_height(cImage))
				stride := int(C.cairo_image_surface_get_stride(cImage))
				data := C.cairo_image_surface_get_data(cImage)

				bufSize := height * stride
				buf := C.GoBytes(unsafe.Pointer(data), C.int(bufSize))
				C.cairo_surface_destroy(cImage)

				sizes := make([]byte, 8+8+8)
				binary.BigEndian.PutUint64(sizes, uint64(width))
				binary.BigEndian.PutUint64(sizes[8:], uint64(height))
				binary.BigEndian.PutUint64(sizes[16:], uint64(stride))
				rgbaStream := io.MultiReader(
					bytes.NewBuffer(bgra.RAWBGRA_HEADER),
					bytes.NewBuffer(sizes),
					bytes.NewBuffer(buf),
				)

				imageResult := p.innerParser.Parse(ctx, rgbaStream, "")
				pageImages = append(pageImages, imageResult)
			}
			C.poppler_page_free_image_mapping(imageMappingList)
		}
		pageResult.Images = pageImages

		C.g_object_unref(C.gpointer(page))
		pages = append(pages, pageResult)
	}

	return &PDFParserResult{Pages: pages, Metadata: meta}
}

func (p *PDFParser) ParseStream(ctx context.Context, file io.Reader, path string) chan StreamResult {
	resultChan := make(chan StreamResult)
	go func() {
		defer close(resultChan)
		resultChan <- &PDFParserStreamResult{FullPath: path, CurrentStage: ProgressNew}

		pdfData, err := io.ReadAll(file)
		if err != nil {
			resultChan <- &PDFParserStreamResult{Err: errors.Join(errors.New("failed to read data to the bytes buffer"), err), FullPath: path, CurrentStage: ProgressCompleted}
			return
		}

		gbytes := C.g_bytes_new(C.gconstpointer(unsafe.Pointer(&pdfData[0])), C.size_t(len(pdfData)))
		if gbytes == nil {
			resultChan <- &PDFParserStreamResult{Err: errors.New("failed to create GBytes"), FullPath: path, CurrentStage: ProgressCompleted}
			return
		}
		defer C.g_bytes_unref(gbytes)

		var gErr *C.GError
		doc := C.poppler_document_new_from_bytes(gbytes, nil, &gErr)
		if doc == nil {
			if gErr != nil {
				defer C.g_error_free(gErr)
				resultChan <- &PDFParserStreamResult{Err: errors.Join(ErrBadFile, errors.New(C.GoString((*C.char)(gErr.message)))), FullPath: path, CurrentStage: ProgressCompleted}
				return
			}
			resultChan <- &PDFParserStreamResult{Err: errors.New("unknown error while reading PDF document"), FullPath: path, CurrentStage: ProgressCompleted}
			return
		}
		defer C.g_object_unref(C.gpointer(doc))

		// Get the metadata
		var meta string
		metaCStr := C.poppler_document_get_metadata(doc)
		if metaCStr != nil {
			meta = C.GoString(metaCStr)
			C.g_free(C.gpointer(metaCStr))
		}

		var pages []PDFParserStreamResultPage

		n_pages := int(C.poppler_document_get_n_pages(doc))
		for pageIndex := range n_pages {
			percentageDone := uint8(math.Round(float64(pageIndex) / float64(n_pages) * 100))
			resultChan <- &PDFParserStreamResult{FullPath: path, CurrentStage: ProgressUpdate, CurrentProgress: percentageDone}

			page := C.poppler_document_get_page(doc, C.int(pageIndex))
			if page == nil {
				continue
			}
			pageResult := PDFParserStreamResultPage{}

			// Get all the text from the page
			textC := C.poppler_page_get_text(page)
			if unsafe.Pointer(textC) != nil {
				text := C.GoString(textC)
				pageResult.Text = text
				C.g_free(C.gpointer(textC))
			}

			// Get all the images on the page
			var pageImages []StreamResult
			imageMappingList := C.poppler_page_get_image_mapping(page)
			if imageMappingList != nil {
				for l := imageMappingList; l != nil; l = l.next {
					mapping := (*C.PopplerImageMapping)(l.data)
					cImage := C.poppler_page_get_image(page, mapping.image_id)
					if cImage == nil {
						continue
					}

					format := C.cairo_image_surface_get_format(cImage)
					if format != C.CAIRO_FORMAT_ARGB32 && format != C.CAIRO_FORMAT_RGB24 {
						C.cairo_surface_destroy(cImage)
						continue
					}

					width := int(C.cairo_image_surface_get_width(cImage))
					height := int(C.cairo_image_surface_get_height(cImage))
					stride := int(C.cairo_image_surface_get_stride(cImage))
					data := C.cairo_image_surface_get_data(cImage)

					bufSize := height * stride
					buf := C.GoBytes(unsafe.Pointer(data), C.int(bufSize))
					C.cairo_surface_destroy(cImage)

					sizes := make([]byte, 8+8+8)
					binary.BigEndian.PutUint64(sizes, uint64(width))
					binary.BigEndian.PutUint64(sizes[8:], uint64(height))
					binary.BigEndian.PutUint64(sizes[16:], uint64(stride))
					rgbaStream := io.MultiReader(
						bytes.NewBuffer(bgra.RAWBGRA_HEADER),
						bytes.NewBuffer(sizes),
						bytes.NewBuffer(buf),
					)

					var lastResult StreamResult
					parseProgress := p.innerParser.ParseStream(ctx, rgbaStream, pathlib.Join(path, fmt.Sprintf("image_%d_%d", int(pageIndex), len(pageImages))))
					for progress := range parseProgress {
						lastResult = progress
						resultChan <- &PDFParserStreamResult{FullPath: path, CurrentStage: ProgressUpdate, CurrentProgress: percentageDone, Subfile: progress}
					}

					if lastResult.Error() == nil {
						pageImages = append(pageImages, lastResult)
					}
				}
				C.poppler_page_free_image_mapping(imageMappingList)
			}
			pageResult.Images = pageImages

			C.g_object_unref(C.gpointer(page))
			pages = append(pages, pageResult)
		}
		resultChan <- &PDFParserStreamResult{FullPath: path, CurrentStage: ProgressCompleted, CurrentProgress: 100, Metadata: meta, Pages: pages}
	}()
	return resultChan
}

type PDFParserResultPage struct {
	Text   string   `json:"text"`
	Images []Result `json:"images"`
}

type PDFParserResult struct {
	FullPath string                `json:"path"`
	Metadata string                `json:"metadata"`
	Pages    []PDFParserResultPage `json:"pages"`
	Err      error                 `json:"error"`
}

func (r *PDFParserResult) Path() string {
	return r.FullPath
}

func (r *PDFParserResult) String() string {
	var result strings.Builder

	if r.Metadata != "" {
		result.WriteString("------ Metadata %d ------\n")
		result.WriteString(r.Metadata)
		result.WriteString("\n")
	}

	for pageIndex, page := range r.Pages {
		result.WriteString(fmt.Sprintf("------ Page %d ------\n", pageIndex))
		result.WriteString(page.Text)
		for imageId, image := range page.Images {
			if image.Error() == nil {
				result.WriteString("\n")
				result.WriteString(fmt.Sprintf("--- Image %d OCR: ---\n", imageId))
				result.WriteString(image.String())
			}
		}
		result.WriteString("\n")
	}

	return result.String()
}

func (r *PDFParserResult) Error() error {
	return r.Err
}

func (r *PDFParserResult) Subfiles() []Result {
	return nil
}

type PDFParserStreamResultPage struct {
	Text   string         `json:"text"`
	Images []StreamResult `json:"images"`
}

type PDFParserStreamResult struct {
	FullPath        string                      `json:"path"`
	Text            string                      `json:"text"`
	CurrentStage    ParseProgressStage          `json:"stage"`
	CurrentProgress uint8                       `json:"progress"`
	Metadata        string                      `json:"metadata"`
	Pages           []PDFParserStreamResultPage `json:"pages"`
	Subfile         StreamResult                `json:"subfile"`
	Err             error                       `json:"error"`
}

func (r *PDFParserStreamResult) Path() string {
	return r.FullPath
}

func (r *PDFParserStreamResult) Stage() ParseProgressStage {
	return r.CurrentStage
}

func (r *PDFParserStreamResult) Progress() uint8 {
	return r.CurrentProgress
}

func (r *PDFParserStreamResult) SubResult() StreamResult {
	return r.Subfile
}

func (r *PDFParserStreamResult) String() string {
	var result strings.Builder

	if r.Metadata != "" {
		result.WriteString("------ Metadata %d ------\n")
		result.WriteString(r.Metadata)
		result.WriteString("\n")
	}

	for pageIndex, page := range r.Pages {
		result.WriteString(fmt.Sprintf("------ Page %d ------\n", pageIndex))
		result.WriteString(page.Text)
		for imageId, image := range page.Images {
			if image.Error() == nil {
				result.WriteString("\n")
				result.WriteString(fmt.Sprintf("--- Image %d OCR: ---\n", imageId))
				result.WriteString(image.String())
			}
		}
		result.WriteString("\n")
	}

	return result.String()
}

func (r *PDFParserStreamResult) Error() error {
	return r.Err
}
