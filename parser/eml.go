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

func (p *EMLParser) ParseStream(ctx context.Context, file io.Reader, path string) chan StreamResult {
	resultChan := make(chan StreamResult)
	go func() {
		defer close(resultChan)
		resultChan <- &EMLParserStreamResult{FullPath: path, CurrentStage: ProgressNew}

		mailReader, err := mail.CreateReader(file)
		if err != nil {
			resultChan <- &EMLParserStreamResult{FullPath: path, CurrentStage: ProgressCompleted, Err: errors.Join(ErrBadFile, err)}
			return
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
				resultChan <- &EMLParserStreamResult{FullPath: path, CurrentStage: ProgressCompleted, Err: errors.Join(ErrBadFile, errors.New("error while reading email part"), err)}
				return
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
				partParse := p.innerParser.ParseStream(ctx, part.Body, pathlib.Join(path, filename))
				var lastProgress StreamResult
				for progress := range partParse {
					lastProgress = progress
					resultChan <- &EMLParserStreamResult{
						FullPath:          path,
						CurrentStage:      ProgressUpdate,
						CurrentPartHeader: part.Header,
						CurrentPart:       progress,
					}
				}

				if lastProgress.Error() != nil || disposition != "attachment" {
					if result.Text != "" {
						result.Text += "\n"
					}
					result.Text += fmt.Sprintf("--- Inline attachment begin: %s ---\n", lastProgress.Path())
					result.Text += lastProgress.String()
					result.Text += fmt.Sprintf("--- Inline attachment end: %s ---\n", lastProgress.Path())
				}
			}
		}

		resultChan <- &EMLParserStreamResult{
			FullPath:     path,
			Text:         result.String(),
			CurrentStage: ProgressCompleted,
		}
	}()
	return resultChan
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
	FullPath          string             `json:"path"`
	Text              string             `json:"text"`
	CurrentStage      ParseProgressStage `json:"stage"`
	CurrentPartHeader mail.PartHeader    `json:"subResultHeader"`
	CurrentPart       StreamResult       `json:"subResult"`
	Err               error              `json:"error"`
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
	return ""
}

func (r *EMLParserStreamResult) Error() error {
	return r.Err
}
