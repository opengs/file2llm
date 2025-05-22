package parser

import (
	"archive/tar"
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	pathlib "path"
)

type TARParser struct {
	innerParser Parser
}

func NewTARParser(innerParser Parser) *TARParser {
	return &TARParser{
		innerParser: innerParser,
	}
}

func (p *TARParser) SupportedMimeTypes() []string {
	return []string{"application/x-tar"}
}

func (p *TARParser) Parse(ctx context.Context, file io.Reader, path string) Result {
	result := &TARParserResult{
		FullPath: path,
	}

	reader := tar.NewReader(file)
	for {
		header, err := reader.Next()
		if err != nil {
			if err == io.EOF {
				break
			}

			return &TARParserResult{Err: errors.Join(ErrBadFile, err), FullPath: path}
		}

		subfileResult := p.innerParser.Parse(ctx, reader, pathlib.Join(path, header.Name))
		result.SubfilesResults = append(result.SubfilesResults, subfileResult)
	}

	return result
}

func (p *TARParser) ParseStream(ctx context.Context, file io.Reader, path string) chan StreamResult {
	resultChan := make(chan StreamResult)
	go func() {
		defer close(resultChan)
		resultChan <- &TARParserStreamResult{FullPath: path, CurrentStage: ProgressNew}

		reader := tar.NewReader(file)
		for {
			if ctx.Err() != nil {
				resultChan <- &TARParserStreamResult{Err: errors.Join(errors.New("parsing cancelled due to context error"), ctx.Err()), FullPath: path, CurrentStage: ProgressCompleted}
				return
			}

			header, err := reader.Next()
			if err != nil {
				if err == io.EOF {
					break
				}

				resultChan <- &TARParserStreamResult{Err: errors.Join(ErrBadFile, err), FullPath: path, CurrentStage: ProgressCompleted}
			}

			parseStream := p.innerParser.ParseStream(ctx, reader, pathlib.Join(path, header.Name))
			for progress := range parseStream {
				resultChan <- &TARParserStreamResult{FullPath: path, CurrentStage: ProgressUpdate, CurrentSubfile: progress}
			}
		}

		resultChan <- &TARParserStreamResult{FullPath: path, CurrentStage: ProgressCompleted}
	}()
	return resultChan
}

type TARParserResult struct {
	FullPath        string   `json:"path"`
	SubfilesResults []Result `json:"subfiles"`
	Err             error    `json:"error"`
}

func (r *TARParserResult) Path() string {
	return r.FullPath
}

func (r *TARParserResult) String() string {
	var result strings.Builder

	for _, subfile := range r.SubfilesResults {
		if subfile.Error() != nil {
			continue
		}

		result.WriteString(fmt.Sprintf("------ File %s ------", subfile.Path()))
		result.WriteString(subfile.String())
		result.WriteString("\n")
	}

	return result.String()
}

func (r *TARParserResult) Error() error {
	return r.Err
}

func (r *TARParserResult) Subfiles() []Result {
	return r.SubfilesResults
}

type TARParserStreamResult struct {
	FullPath       string             `json:"path"`
	CurrentStage   ParseProgressStage `json:"stage"`
	CurrentSubfile StreamResult       `json:"subResult"`
	Err            error              `json:"error"`
}

func (r *TARParserStreamResult) Path() string {
	return r.FullPath
}

func (r *TARParserStreamResult) Stage() ParseProgressStage {
	return r.CurrentStage
}

func (r *TARParserStreamResult) Progress() uint8 {
	return 0
}

func (r *TARParserStreamResult) SubResult() StreamResult {
	return r.CurrentSubfile
}

func (r *TARParserStreamResult) String() string {
	return ""
}

func (r *TARParserStreamResult) Error() error {
	return r.Err
}
