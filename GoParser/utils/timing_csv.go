package utils

import (
	"GoParser/model"
	"encoding/csv"
	"fmt"
	"os"
)

type TimingCSVWriter struct {
	writer *csv.Writer
	file   *os.File
}

func NewTimingCSVWriter(path string) (*TimingCSVWriter, error) {
	f, err := os.Create(path)
	if err != nil {
		return nil, fmt.Errorf("failed to create timing CSV: %w", err)
	}
	w := csv.NewWriter(f)
	header := []string{
		"repository",
		"fetch_files_ms",
		"parse_ast_ms",
		"topo_sort_ms",
		"type_check_ms",
		"collect_symbols_ms",
		"check_runner_ms",
		"total_analysis_ms",
		"total_ms",
	}
	if err := w.Write(header); err != nil {
		f.Close()
		return nil, err
	}
	w.Flush()
	return &TimingCSVWriter{writer: w, file: f}, nil
}

func (t *TimingCSVWriter) WriteRow(timing model.AnalysisTiming) error {
	ms := func(d interface{ Milliseconds() int64 }) string {
		return fmt.Sprintf("%d", d.Milliseconds())
	}
	row := []string{
		timing.Repository,
		ms(timing.FetchFiles),
		ms(timing.ParseAST),
		ms(timing.TopoSort),
		ms(timing.TypeCheck),
		ms(timing.CollectSymbols),
		ms(timing.CheckRunner),
		ms(timing.TotalAnalysis),
		ms(timing.Total),
	}
	if err := t.writer.Write(row); err != nil {
		return err
	}
	t.writer.Flush()
	return t.writer.Error()
}

func (t *TimingCSVWriter) Close() error {
	t.writer.Flush()
	return t.file.Close()
}