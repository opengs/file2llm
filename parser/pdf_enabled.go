package parser

/*
 #cgo pkg-config: poppler-glib
 #cgo pkg-config: gdk-pixbuf-2.0
 #include <poppler.h>
 #include <gdk-pixbuf/gdk-pixbuf.h>
 #include <stdlib.h>
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

			pixbuf := (*C.GdkPixbuf)(unsafe.Pointer(image))
			dataPtr := C.gdk_pixbuf_get_pixels(pixbuf)
			dataLen := int(C.gdk_pixbuf_get_byte_length(pixbuf))
			goImageBytes := C.GoBytes(unsafe.Pointer(dataPtr), C.int(dataLen))
			C.g_object_unref(C.gpointer(image))

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
