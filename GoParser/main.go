package main

import (
	"GoParser/model"
	"fmt"
	"log"
	"path/filepath"
	"time"

	utils "GoParser/utils"
)

type AnalysisJob struct {
	Name       string
	FetchFiles func() ([]model.FileInfo, error)
}

func main() {
	config, err := utils.SetupEnvironment()
	if err != nil {
		log.Fatalf("Failed to set up environment: %v", err)
	}

	dbs, err := utils.CreateDatabases(config)
	if err != nil {
		log.Fatalf("Failed to create databases: %v", err)
	}
	defer func() {
		if err := dbs.Close(); err != nil {
			log.Fatalf("Failed to close databases: %v", err)
		}
	}()

	timingWriter, err := utils.NewTimingCSVWriter("../output/timing.csv")
	if err != nil {
		log.Fatalf("Failed to create timing CSV: %v", err)
	}
	defer timingWriter.Close()

	astAnalyzer := NewASTAnalyzer()

	if config.EnableTypeInference {
		log.Printf("Type inference analysis ENABLED (Erweiterung 4)")
	} else {
		log.Printf("Type inference analysis DISABLED (use ENABLE_TYPE_INFERENCE=true to enable)")
	}

	// Build the list of analysis jobs depending on the mode.
	var jobs []AnalysisJob

	if config.LocalProject != "" {
		log.Printf("Running in LOCAL mode for project: %s", config.LocalProject)
		projectName := "local/" + filepath.Base(config.LocalProject)
		jobs = []AnalysisJob{{
			Name: projectName,
			FetchFiles: func() ([]model.FileInfo, error) {
				files, err := utils.FetchLocalGoFiles(config.LocalProject)
				if err == nil {
					log.Printf("Found %d files in local project", len(files))
				}
				return files, err
			},
		}}
	} else {
		log.Printf("Running in GITHUB mode")
		entries, err := utils.GetOwnerAndRepo(config.InputCSVPath)
		if err != nil {
			log.Fatalf("Failed to read CSV file: %v", err)
		}
		for _, repo := range entries {
			owner, repoName := repo[0], repo[1]
			jobs = append(jobs, AnalysisJob{
				Name: owner + "/" + repoName,
				FetchFiles: func() ([]model.FileInfo, error) {
					return utils.FetchGoFilesList(owner, repoName, config.Token)
				},
			})
		}
	}

	utils.PrintCSVHeader()

	var results []model.GenericCounters
	for _, job := range jobs {
		counters, instantiations, err := processJob(job, astAnalyzer, dbs, config.EnableTypeInference, timingWriter)
		if err != nil {
			log.Println(err)
			continue
		}
		if config.EnableTypeInference && len(instantiations) > 0 && config.LocalProject != "" {
			utils.PrintInstantiationSummary(job.Name, instantiations)
		}
		log.Printf("Finished: %s", job.Name)
		results = append(results, counters)
	}

	if config.LocalProject != "" {
		if len(results) > 0 {
			utils.PrintCountersSummary(results[0], "Counter for local project")
		}
	} else {
		summary := utils.ComputeCrossRepoAggregation(results)
		utils.PrintCountersSummary(summary, "Counter over every Repository")
	}
}

func processJob(job AnalysisJob, analyzer ASTAnalyzer, dbs *utils.DatabaseSet, enableTypeInference bool, timingWriter *utils.TimingCSVWriter) (model.GenericCounters, model.InstantiationData, error) {
	tFetch := time.Now()
	files, err := job.FetchFiles()
	if err != nil {
		return model.GenericCounters{}, nil, err
	}
	fetchDuration := time.Since(tFetch)

	counters, instantiations, timing, err := analyzer.AnalyzeProject(files, enableTypeInference)
	if err != nil {
		return model.GenericCounters{}, nil, fmt.Errorf("analysis failed: %w", err)
	}
	counters.Repository = job.Name

	timing.Repository = job.Name
	timing.FetchFiles = fetchDuration
	timing.Total = timing.FetchFiles + timing.TotalAnalysis

	if timingWriter != nil {
		if err := timingWriter.WriteRow(timing); err != nil {
			log.Printf("Warning: failed to write timing row: %v", err)
		}
	}

	utils.PrintCSVRow(job.Name, counters)

	if err := dbs.Generics.AddGenericCountersEntry(counters); err != nil {
		return counters, instantiations, fmt.Errorf("failed to add entry to database: %w", err)
	}

	if dbs.Instantiations != nil && len(instantiations) > 0 {
		if err := dbs.Instantiations.AddInstantiationData(job.Name, instantiations); err != nil {
			return counters, instantiations, fmt.Errorf("failed to add instantiation data to database: %w", err)
		}
	}

	return counters, instantiations, nil
}