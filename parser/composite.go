package parser

import (
	"bytes"
	"context"
	"errors"
	"io"

	"github.com/gabriel-vasile/mimetype"
)

type CompositeParser struct {
	parsers      []Parser
	mimeToParser map[string]Parser
}

func NewCompositeParser(parsers ...Parser) *CompositeParser {
	mimeToParser := make(map[string]Parser, 32)
	for _, parser := range parsers {
		for _, mt := range parser.SupportedMimeTypes() {
			mimeToParser[mt] = parser
		}
	}

	return &CompositeParser{
		parsers:      parsers,
		mimeToParser: mimeToParser,
	}
}

func (p *CompositeParser) AddParsers(parsers ...Parser) {
	for _, parser := range parsers {
		for _, mt := range parser.SupportedMimeTypes() {
			p.mimeToParser[mt] = parser
		}
	}
	p.parsers = append(p.parsers, parsers...)
}

func (p *CompositeParser) SupportedMimeTypes() []string {
	mimeTypes := make([]string, 0, len(p.mimeToParser))
	for k := range p.mimeToParser {
		mimeTypes = append(mimeTypes, k)
	}

	return mimeTypes
}

func (p *CompositeParser) Parse(ctx context.Context, file io.Reader) Result {
	mimeBlock := make([]byte, 1024)
	readed, err := io.ReadFull(file, mimeBlock)
	if err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
		return &CompositeParserResult{Err: errors.Join(errors.New("failed to read file to determine mime type"), err)}
	}

	mime := mimetype.Detect(mimeBlock[:readed])
	if parser, ok := p.mimeToParser[mime.String()]; ok {
		result := parser.Parse(ctx, io.MultiReader(bytes.NewBuffer(mimeBlock[:readed]), file))
		return &CompositeParserResult{Inner: result, MimeType: mime.String()}
	}

	return &CompositeParserResult{Err: &ErrMimeTypeNotSupported{MimeType: mime}, MimeType: mime.String()}
}

type CompositeParserResult struct {
	Err      error  `json:"error"`
	MimeType string `json:"mimeType"`
	Inner    Result `json:"inner"`
}

func (r *CompositeParserResult) String() string {
	if r.Inner != nil {
		return r.Inner.String()
	}
	return ""
}

func (r *CompositeParserResult) Error() error {
	if r.Inner != nil {
		return r.Inner.Error()
	}
	return r.Err
}

func (r *CompositeParserResult) Componets() []Result {
	if r.Inner != nil {
		return r.Inner.Componets()
	}
	return nil
}
