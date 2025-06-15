package parser

import (
	"context"
	"errors"
	"fmt"
	"io"
	"mime"
	pathlib "path"
	"path/filepath"
	"strings"

	"github.com/emersion/go-message"
	_ "github.com/emersion/go-message/charset"
	"github.com/emersion/go-message/mail"
)

// Parses `message/rfc822` files (.eml)
type EMLParser struct {
	innerParser Parser
}

func NewEMLParser(innerParser Parser) *EMLParser {
	return &EMLParser{
		innerParser: innerParser,
	}
}

func (p *EMLParser) SupportedMimeTypes() []string {
	return []string{"message/rfc822"}
}

func (p *EMLParser) Parse(ctx context.Context, file io.Reader, path string) Result {
	mailReader, err := mail.CreateReader(file)
	if err != nil {
		return &EMLParserResult{
			Err:      errors.Join(ErrBadFile, err),
			FullPath: path,
		}
	}

	result := EMLParserResult{
		Headers:  mailReader.Header.Map(),
		FullPath: path,
	}

	partID := -1
	for {
		partID += 1
		part, err := mailReader.NextPart()
		if err == io.EOF {
			break
		} else if err != nil {
			return &EMLParserResult{
				Err:      errors.Join(ErrBadFile, errors.New("error while reading email part"), err),
				FullPath: path,
			}
		}

		contentType, ctParams, _ := mime.ParseMediaType(part.Header.Get("Content-Type"))
		disposition, dispParams, _ := mime.ParseMediaType(part.Header.Get("Content-Disposition"))

		if contentType == "text/plain" {
			body, _ := io.ReadAll(part.Body)
			if result.Text != "" {
				result.Text += "\n"
			}
			result.Text += string(body)
		} else {
			filename := p.getFileName(ctParams, dispParams, partID)
			r := p.innerParser.Parse(ctx, part.Body, pathlib.Join(path, filename))
			if r.Error() != nil || disposition == "attachment" {
				result.Attachments = append(result.Attachments, r)
			} else {
				if result.Text != "" {
					result.Text += "\n"
				}
				result.Text += fmt.Sprintf("--- Inline attachment begin: %s ---\n", r.Path())
				result.Text += r.String()
				result.Text += fmt.Sprintf("--- Inline attachment end: %s ---\n", r.Path())
			}
		}
	}

	return &result
}

func (p *EMLParser) ParseStream(ctx context.Context, file io.Reader, path string) StreamResultIterator {
	return &EMLStreamResultIterator{
		emlParser: p,
		ctx:       ctx,
		file:      file,
		path:      path,
	}
}

func (p *EMLParser) getFileName(ctParams, dispParams map[string]string, partID int) string {
	if name := dispParams["filename"]; name != "" {
		return filepath.Base(name)
	}
	if name := ctParams["name"]; name != "" {
		return filepath.Base(name)
	}
	return fmt.Sprintf("ext_%d", partID)
}

type EMLStreamResultIterator struct {
	emlParser *EMLParser
	ctx       context.Context
	file      io.Reader
	path      string

	completed           bool
	initialized         bool
	initializationError error
	reader              *mail.Reader
	part                *mail.Part
	partDisposition     string
	partIndex           int
	partParse           StreamResultIterator

	current StreamResult
}

func (i *EMLStreamResultIterator) Next(ctx context.Context) bool {
	if i.completed {
		i.current = nil
		return false
	}

	if i.initializationError != nil {
		i.completed = true
		i.current = &EMLParserStreamResult{
			FullPath: i.path,
			Err:      i.initializationError,
		}
		return true
	}

	if i.reader == nil {
		msg, err := message.Read(i.file)
		if err != nil {
			i.initializationError = err
			i.current = &EMLParserStreamResult{
				FullPath:     i.path,
				CurrentStage: ProgressNew,
			}
			return true
		}

		i.reader = mail.NewReader(msg)
		i.current = &EMLParserStreamResult{
			FullPath:     i.path,
			CurrentStage: ProgressNew,
			Headers:      i.reader.Header.Map(),
		}
		return true
	}

	if i.part == nil {
		i.partIndex += 1
		part, err := i.reader.NextPart()
		if err != nil {
			if err == io.EOF {
				i.completed = true
				i.current = &EMLParserStreamResult{
					FullPath:     i.path,
					CurrentStage: ProgressCompleted,
				}
				return true
			}

			i.completed = true
			i.current = &EMLParserStreamResult{
				FullPath:     i.path,
				CurrentStage: ProgressCompleted,
				Err:          errors.Join(errors.New("failed to get next part from email"), err),
			}
			return true
		}

		contentType, ctParams, _ := mime.ParseMediaType(part.Header.Get("Content-Type"))
		disposition, dispParams, _ := mime.ParseMediaType(part.Header.Get("Content-Disposition"))
		i.partDisposition = disposition

		if contentType == "text/plain" {
			body, err := io.ReadAll(part.Body)
			if err != nil {
				i.completed = true
				i.current = &EMLParserStreamResult{
					FullPath:     i.path,
					CurrentStage: ProgressCompleted,
					Err:          errors.Join(errors.New("failed to read part body"), err),
				}
				return true
			}

			i.current = &EMLParserStreamResult{
				FullPath:          i.path,
				CurrentStage:      ProgressUpdate,
				CurrentPartHeader: i.part.Header,
				Text:              string(body),
			}
			i.part = nil
			return true
		} else {
			filename := i.emlParser.getFileName(ctParams, dispParams, i.partIndex)
			i.partParse = i.emlParser.innerParser.ParseStream(ctx, part.Body, pathlib.Join(i.path, filename))
		}
	}

	if i.partParse.Next(ctx) {
		if i.partDisposition != "attachment" {
			i.current = &EMLParserStreamResult{
				FullPath:     i.path,
				CurrentStage: ProgressUpdate,
				Text:         i.partParse.Current().String(),
			}
			if i.partParse.Current().Error() != nil {
				i.current = &EMLParserStreamResult{
					FullPath:     i.path,
					CurrentStage: ProgressCompleted,
					Text:         i.partParse.Current().String(),
					Err:          errors.Join(fmt.Errorf("failed to parse embeded part %s of the email", i.partParse.Current().Path()), i.partParse.Current().Error()),
				}
				i.completed = true
			}
		} else {
			i.current = &EMLParserStreamResult{
				FullPath:          i.path,
				CurrentStage:      ProgressUpdate,
				CurrentPartHeader: i.part.Header,
				CurrentPart:       i.partParse.Current(),
			}
		}
		return true
	} else {
		i.partParse.Close()
		i.part = nil
		return i.Next(ctx)
	}
}

func (i *EMLStreamResultIterator) Current() StreamResult {
	return i.current
}

func (i *EMLStreamResultIterator) Close() {
	if i.partParse != nil {
		i.partParse.Close()
		i.partParse = nil
	}
}

type EMLParserResult struct {
	FullPath    string              `json:"path"`
	Headers     map[string][]string `json:"headers"`
	Text        string              `json:"text"`
	Err         error               `json:"error"`
	Attachments []Result            `json:"attachments"`
}

func (r *EMLParserResult) Path() string {
	return r.FullPath
}

func (r *EMLParserResult) String() string {
	var result strings.Builder

	if len(r.Headers) > 0 {
		result.WriteString("----- Headers -----\n")
		for key, val := range r.Headers {
			result.WriteString(fmt.Sprintf("%s: %s\n", key, strings.Join(val, ", ")))
		}
	}

	if r.Text != "" {
		result.WriteString("----- Body -----\n")
		result.WriteString(r.Text)
	}

	if len(r.Attachments) != 0 {
		result.WriteString("----- Attachments -----\n")
		for _, attachment := range r.Attachments {
			result.WriteString(fmt.Sprintf("--- Attachment %s ---\n", attachment.Path()))
			result.WriteString(attachment.String())
			result.WriteString("\n")
		}
	}

	return result.String()
}

func (r *EMLParserResult) Error() error {
	return r.Err
}

func (r *EMLParserResult) Subfiles() []Result {
	return r.Attachments
}

type EMLParserStreamResult struct {
	FullPath          string              `json:"path"`
	Text              string              `json:"text"`
	CurrentStage      ParseProgressStage  `json:"stage"`
	Headers           map[string][]string `json:"headers"`
	CurrentPartHeader mail.PartHeader     `json:"subResultHeader"`
	CurrentPart       StreamResult        `json:"subResult"`
	Err               error               `json:"error"`
}

func (r *EMLParserStreamResult) Path() string {
	return r.FullPath
}

func (r *EMLParserStreamResult) Stage() ParseProgressStage {
	return r.CurrentStage
}

func (r *EMLParserStreamResult) Progress() uint8 {
	return 0
}

func (r *EMLParserStreamResult) SubResult() StreamResult {
	return r.CurrentPart
}

func (r *EMLParserStreamResult) String() string {
	var result strings.Builder

	if len(r.Headers) != 0 {
		result.WriteString("------ Headers start------\n")
		for header, vals := range r.Headers {
			result.WriteString(header)
			result.WriteString(": ")
			result.WriteString(strings.Join(vals, ", "))
			result.WriteString("\n")
		}
		result.WriteString("------ Headers end------\n\n")
	}

	if len(r.Text) != 0 {
		result.WriteString(r.Text)
	}

	return result.String()
}

func (r *EMLParserStreamResult) Error() error {
	return r.Err
}
