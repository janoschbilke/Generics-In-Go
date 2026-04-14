package main

import (
	"GoParser/database"
	"GoParser/model"
	"fmt"
	"log"
	"path/filepath"

	utils "GoParser/utils"
)

// AnalysisJob represents a single unit of analysis work: a named project and a way to fetch its files. (In preparation for upcoming parallelization)
type AnalysisJob struct {
	Name       string
	FetchFiles func() ([]string, error)
}

func main() {
	config, err := utils.SetupEnvironment()
	if err != nil {
		log.Fatalf("Failed to set up environment: %v", err)
	}

	genericsDatabase, err := utils.CreateDatabase(config)
	if err != nil {
		log.Fatalf("Failed to create database: %v", err)
	}
	defer func() {
		if err := genericsDatabase.Close(); err != nil {
			log.Fatalf("Failed to close genericsDatabase: %v", err)
		}
	}()

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
			FetchFiles: func() ([]string, error) {
				files, err := utils.FetchLocalGoFiles(config.LocalProject)
				if err == nil {
					log.Printf("Found %d .go files in local project", len(files))
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
				FetchFiles: func() ([]string, error) {
					return utils.FetchGoFilesList(owner, repoName, config.Token)
				},
			})
		}
	}

	utils.PrintCSVHeader()

	var results []model.GenericCounters
	for _, job := range jobs {
		counters, err := processJob(job, astAnalyzer, genericsDatabase, config.EnableTypeInference)
		if err != nil {
			log.Println(err)
			continue
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

func analyzeFiles(files []string, analyzer ASTAnalyzer, enableTypeInference bool) model.GenericCounters {
	counters := model.GenericCounters{}
	for _, file := range files {
		counts, err := analyzer.AnalyzeFileWithConfig(file, enableTypeInference)
		if err != nil {
			log.Println("Error:", err)
		} else {
			aggregateCounters(&counters, counts)
		}
	}
	return counters
}

// processJob fetches files for a job, analyzes them, prints a CSV row, and saves the result to the DB, it returns the aggregated counters for the job so callers can compute summaries
func processJob(job AnalysisJob, analyzer ASTAnalyzer, db database.GenericsDatabase, enableTypeInference bool) (model.GenericCounters, error) {
	files, err := job.FetchFiles()
	if err != nil {
		return model.GenericCounters{}, err
	}

	counters := analyzeFiles(files, analyzer, enableTypeInference)
	counters.Repository = job.Name

	utils.PrintCSVRow(job.Name, counters)

	if err := db.AddGenericCountersEntry(counters); err != nil {
		return counters, fmt.Errorf("failed to add entry to database: %w", err)
	}

	return counters, nil
}

func aggregateCounters(target *model.GenericCounters, source model.GenericCounters) {
	target.FuncTotal += source.FuncTotal
	target.FuncGeneric += source.FuncGeneric
	target.MethodTotal += source.MethodTotal
	target.MethodWithGenericReceiver += source.MethodWithGenericReceiver
	target.MethodWithGenericReceiverTrivialTypeBound += source.MethodWithGenericReceiverTrivialTypeBound
	target.MethodWithGenericReceiverNonTrivialTypeBound += source.MethodWithGenericReceiverNonTrivialTypeBound
	target.StructTotal += source.StructTotal
	target.StructGeneric += source.StructGeneric
	target.StructGenericBound += source.StructGenericBound
	target.StructAsTypeBound += source.StructAsTypeBound
	target.TypeDecl += source.TypeDecl
	target.GenericTypeDecl += source.GenericTypeDecl
	target.GenericTypeSet += source.GenericTypeSet

	target.GenericFuncInstantiationExplicit += source.GenericFuncInstantiationExplicit
	target.GenericFuncInstantiationInferred += source.GenericFuncInstantiationInferred
	target.GenericTypeInstantiationExplicit += source.GenericTypeInstantiationExplicit
	target.GenericTypeInstantiationInferred += source.GenericTypeInstantiationInferred
}
