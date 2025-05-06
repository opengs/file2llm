//go:build file2llm_feature_pdf || test

package parser

/*
 #cgo pkg-config: poppler-glib cairo
 #include <poppler.h>
 #include <cairo.h>
 #include <cairo-pdf.h>
 #include <stdlib.h>

typedef struct {
	unsigned char *current_position;
	unsigned char *end_of_array;
} png_stream_to_byte_array_closure_t;

static cairo_status_t write_png_stream_to_byte_array (void *in_closure, const unsigned char *data, unsigned int length) {
	png_stream_to_byte_array_closure_t *closure = (png_stream_to_byte_array_closure_t *) in_closure;

	if ((closure->current_position + length) > (closure->end_of_array)) {
		return CAIRO_STATUS_WRITE_ERROR;
	}

	memcpy (closure->current_position, data, length);
	closure->current_position += length;

	return CAIRO_STATUS_SUCCESS;
}

cairo_status_t cairo_surface_to_png_bytes(cairo_surface_t *surface, unsigned char* buffer, size_t buffer_size, size_t* len) {
    png_stream_to_byte_array_closure_t closure;

    closure.current_position = buffer;
    closure.end_of_array = buffer + buffer_size;

    cairo_status_t status = cairo_surface_write_to_png_stream(surface, write_png_stream_to_byte_array, &closure);
    if (status != CAIRO_STATUS_SUCCESS) {
        return status;
    }

    *len = closure.current_position - buffer; // how many bytes were written
    return CAIRO_STATUS_SUCCESS;
}


*/
import "C"
import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"unsafe"
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

func (p *PDFParser) Parse(ctx context.Context, file io.Reader) Result {
	pdfData, err := io.ReadAll(file)
	if err != nil {
		return &PDFParserResult{Err: errors.Join(errors.New("failed to read data to the bytes buffer"), err)}
	}

	cData := (*C.char)(unsafe.Pointer(&pdfData[0]))
	dataLength := C.int(len(pdfData))

	var gErr *C.GError
	doc := C.poppler_document_new_from_data(cData, dataLength, nil, &gErr)
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

	n_pages := C.poppler_document_get_n_pages(doc)
	for pageIndex := range n_pages {
		page := C.poppler_document_get_page(doc, pageIndex)
		if page == nil {
			continue
		}
		pageResult := PDFParserResultPage{}

		// Get all the text from the page
		textC := C.poppler_page_get_text(page)
		text := C.GoString((*C.char)(textC))
		pageResult.Text = text
		C.g_free(C.gpointer(textC))

		// Get all the images on the page
		var pageImages []Result
		imageMappingList := C.poppler_page_get_image_mapping(page)
		for l := imageMappingList; l != nil; l = l.next {
			mapping := (*C.PopplerImageMapping)(l.data)
			image := C.poppler_page_get_image(page, mapping.image_id)
			if image == nil {
				continue
			}

			bufferSize := 8 * 1024 * 1024 // 8 MB
			buffer := make([]byte, bufferSize)

			var writtenLength C.size_t
			status := C.cairo_surface_to_png_bytes(image, (*C.uchar)(unsafe.Pointer(&buffer[0])), C.size_t(bufferSize), &writtenLength)
			C.cairo_surface_destroy(image)
			if status != C.CAIRO_STATUS_SUCCESS {
				continue
			}

			goImageBytes := buffer[:writtenLength]

			imageResult := p.innerParser.Parse(ctx, bytes.NewBuffer(goImageBytes))
			pageImages = append(pageImages, imageResult)
		}
		C.poppler_page_free_image_mapping(imageMappingList)
		pageResult.Images = pageImages

		C.g_object_unref(C.gpointer(page))
		pages = append(pages, pageResult)
	}

	return &PDFParserResult{Pages: pages, Metadata: meta}
}

type PDFParserResultPage struct {
	Text   string   `json:"text"`
	Images []Result `json:"images"`
}

type PDFParserResult struct {
	Metadata string                `json:"metadata"`
	Pages    []PDFParserResultPage `json:"pages"`
	Err      error                 `json:"error"`
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

func (r *PDFParserResult) Componets() []Result {
	return nil
}
