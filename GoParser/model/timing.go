package model

import "time"

type AnalysisTiming struct {
	Repository     string
	FetchFiles     time.Duration
	ParseAST       time.Duration
	TopoSort       time.Duration
	TypeCheck      time.Duration
	CollectSymbols time.Duration
	CheckRunner    time.Duration
	TotalAnalysis  time.Duration
	Total          time.Duration
}