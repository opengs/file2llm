package parser

import (
	"bytes"
	"context"
	"errors"
	"io"

	"github.com/gabriel-vasile/mimetype"
)

type CompositeParser struct {
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
		mimeToParser: mimeToParser,
	}
}

func (p *CompositeParser) AddParsers(parsers ...Parser) {
	for _, parser := range parsers {
		for _, mt := range parser.SupportedMimeTypes() {
			p.mimeToParser[mt] = parser
		}
	}
}

func (p *CompositeParser) SupportedMimeTypes() []string {
	mimeTypes := make([]string, 0, len(p.mimeToParser))
	for k := range p.mimeToParser {
		mimeTypes = append(mimeTypes, k)
	}

	return mimeTypes
}

func (p *CompositeParser) Parse(ctx context.Context, file io.Reader, path string) Result {
	mimeBlock := make([]byte, 1024)
	readed, err := io.ReadFull(file, mimeBlock)
	if err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
		return &CompositeParserResult{Err: errors.Join(errors.New("failed to read file to determine mime type"), err), FullPath: path}
	}

	mime := mimetype.Detect(mimeBlock[:readed])
	if parser, ok := p.mimeToParser[mime.String()]; ok {
		result := parser.Parse(ctx, io.MultiReader(bytes.NewBuffer(mimeBlock[:readed]), file), path)
		return &CompositeParserResult{Inner: result, MimeType: mime.String()}
	}

	return &CompositeParserResult{Err: &ErrMimeTypeNotSupported{MimeType: mime}, MimeType: mime.String(), FullPath: path}
}

func (p *CompositeParser) ParseStream(ctx context.Context, file io.Reader, path string) StreamResultIterator {
	return &CompositeStreamResultIterator{
		compositeParser: p,
		ctx:             ctx,
		file:            file,
		path:            path,
	}
}

type CompositeStreamResultIterator struct {
	compositeParser *CompositeParser
	ctx             context.Context
	file            io.Reader
	path            string

	initialized   bool
	initError     error
	initErrorSent bool
	initResult    StreamResult

	parseStream StreamResultIterator
}

func (i *CompositeStreamResultIterator) Next(ctx context.Context) bool {
	if !i.initialized {
		i.initialized = true
		mimeBlock := make([]byte, 1024)
		readed, err := io.ReadFull(i.file, mimeBlock)
		if err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
			i.initError = errors.Join(errors.New("failed to read file to determine mime type"), err)
			i.initResult = &CompositeParserStreamResult{FullPath: i.path, CurrentStage: ProgressNew}
			return true
		}

		mime := mimetype.Detect(mimeBlock[:readed])
		if parser, ok := i.compositeParser.mimeToParser[mime.String()]; ok {
			i.parseStream = parser.ParseStream(i.ctx, io.MultiReader(bytes.NewBuffer(mimeBlock[:readed]), i.file), i.path)
			return i.parseStream.Next(ctx)
		} else {
			i.initError = &ErrMimeTypeNotSupported{MimeType: mime}
			i.initResult = &CompositeParserStreamResult{FullPath: i.path, CurrentStage: ProgressNew}
			return true
		}
	}

	if i.initError != nil {
		if i.initErrorSent {
			i.initResult = nil
			return false
		}
		i.initErrorSent = true
		i.initResult = &CompositeParserStreamResult{
			FullPath:     i.path,
			CurrentStage: ProgressCompleted,
			Err:          i.initError,
		}
		return true
	}

	return i.parseStream.Next(ctx)
}

func (i *CompositeStreamResultIterator) Current() StreamResult {
	if i.initError != nil {
		return i.initResult
	}
	return i.parseStream.Current()
}

func (i *CompositeStreamResultIterator) Close() {
	if i.parseStream != nil {
		i.parseStream.Close()
	}
}

type CompositeParserResult struct {
	FullPath string `json:"path"`
	Err      error  `json:"error"`
	MimeType string `json:"mimeType"`
	Inner    Result `json:"inner"`
}

func (r *CompositeParserResult) Path() string {
	if r.Inner != nil {
		return r.Inner.Path()
	}
	return r.FullPath
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

func (r *CompositeParserResult) Subfiles() []Result {
	if r.Inner != nil {
		return r.Inner.Subfiles()
	}
	return nil
}

type CompositeParserStreamResult struct {
	FullPath        string             `json:"path"`
	Text            string             `json:"text"`
	MimeType        string             `json:"mimeType"`
	Inner           StreamResult       `json:"inner"`
	CurrentStage    ParseProgressStage `json:"stage"`
	CurrentProgress uint8              `json:"progress"`
	Err             error              `json:"error"`
}

func (r *CompositeParserStreamResult) Path() string {
	if r.Inner != nil {
		return r.Inner.Path()
	}
	return r.FullPath
}

func (r *CompositeParserStreamResult) Stage() ParseProgressStage {
	if r.Inner != nil {
		return r.Inner.Stage()
	}
	return r.CurrentStage
}

func (r *CompositeParserStreamResult) Progress() uint8 {
	if r.Inner != nil {
		return r.Inner.Progress()
	}
	return r.CurrentProgress
}

func (r *CompositeParserStreamResult) SubResult() StreamResult {
	if r.Inner != nil {
		return r.Inner.SubResult()
	}
	return nil
}

func (r *CompositeParserStreamResult) String() string {
	if r.Inner != nil {
		return r.Inner.String()
	}
	return ""
}

func (r *CompositeParserStreamResult) Error() error {
	if r.Inner != nil {
		return r.Inner.Error()
	}
	return r.Err
}
