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

func (p *TARParser) ParseStream(ctx context.Context, file io.Reader, path string) StreamResultIterator {
	return &TARStreamResultIterator{
		ctx:         ctx,
		file:        file,
		path:        path,
		innerParser: p.innerParser,
	}
}

type TARStreamResultIterator struct {
	ctx         context.Context
	file        io.Reader
	path        string
	innerParser Parser

	completed   bool
	reader      *tar.Reader
	parseStream StreamResultIterator
	current     StreamResult
}

func (i *TARStreamResultIterator) Next(ctx context.Context) bool {
	if i.completed {
		i.current = nil
		return false
	}

	if i.reader == nil {
		i.reader = tar.NewReader(i.file)
		i.current = &TARParserStreamResult{
			FullPath:     i.path,
			CurrentStage: ProgressNew,
		}
		return true
	}

	if i.parseStream != nil {
		if i.parseStream.Next(ctx) {
			return true
		} else {
			i.parseStream.Close()
			i.parseStream = nil
		}
	}

	header, err := i.reader.Next()
	if err != nil {
		i.completed = true
		if err == io.EOF {
			i.current = &TARParserStreamResult{FullPath: i.path, CurrentStage: ProgressCompleted}
			return true
		}

		i.current = &TARParserStreamResult{Err: errors.Join(ErrBadFile, err), FullPath: i.path, CurrentStage: ProgressCompleted}
		return true
	}

	i.parseStream = i.innerParser.ParseStream(ctx, i.reader, pathlib.Join(i.path, header.Name))
	return i.Next(ctx)
}

func (i *TARStreamResultIterator) Current() StreamResult {
	return i.current
}

func (i *TARStreamResultIterator) Close() {
	if i.parseStream != nil {
		i.parseStream.Close()
	}
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
