package main

import (
	"fmt"
	"log"
	"path/filepath"

	utils "GoParser/utils"
)

func aggregateCounters(target *GenericCounters, source GenericCounters) {
	target.FuncTotal += source.FuncTotal
	target.FuncGeneric += source.FuncGeneric
	target.MethodTotal += source.MethodTotal
	target.MethodWithGenericReceiver += source.MethodWithGenericReceiver
	target.StructTotal += source.StructTotal
	target.StructGeneric += source.StructGeneric
	target.StructGenericBound += source.StructGenericBound
	target.StructAsTypeBound += source.StructAsTypeBound
	target.TypeDecl += source.TypeDecl
	target.GenericTypeDecl += source.GenericTypeDecl
	target.GenericTypeSet += source.GenericTypeSet
}

func printCountersSummary(counters GenericCounters, title string) {
	fmt.Println()
	fmt.Printf("%s:\n", title)
	fmt.Printf("FuncGeneric: %v\n", counters.FuncGeneric)
	fmt.Printf("MethodWithGenericReceiver: %v\n", counters.MethodWithGenericReceiver)
	fmt.Printf("StructGeneric: %v\n", counters.StructGeneric)
	fmt.Printf("StructGenericNonTrivialBound: %v\n", counters.StructGenericBound)
	fmt.Printf("StructAsTypeBound: %v\n", counters.StructAsTypeBound)
	fmt.Printf("GenericTypeDecl: %v\n", counters.GenericTypeDecl)
	fmt.Printf("GenericTypeSet: %v\n", counters.GenericTypeSet)
}

func printCSVRow(name string, counters GenericCounters) {
	fmt.Printf("%s,%d,%d,%d,%d,%d,%d,%d,%d,%d,%d,%d\n",
		name,
		counters.FuncTotal,
		counters.FuncGeneric,
		counters.MethodTotal,
		counters.MethodWithGenericReceiver,
		counters.StructTotal,
		counters.StructGeneric,
		counters.StructGenericBound,
		counters.StructAsTypeBound,
		counters.TypeDecl,
		counters.GenericTypeDecl,
		counters.GenericTypeSet,
	)
}

func main() {
	config, err := utils.SetupEnvironment()
	if err != nil {
		log.Fatalf("Failed to set up environment: %v", err)
	}

	counterOverEveryRepository := GenericCounters{}

	// Prüfe ob lokaler Modus aktiviert ist
	if config.LocalProject != "" {
		// === LOKALER MODUS ===
		log.Printf("Running in LOCAL mode for project: %s", config.LocalProject)

		files, err := utils.FetchLocalGoFiles(config.LocalProject)
		if err != nil {
			log.Fatalf("Failed to load local files: %v", err)
		}

		log.Printf("Found %d .go files in local project", len(files))

		countersForProject := GenericCounters{}

		// CSV-Header ausgeben
		fmt.Println("Repository,FuncTotal,FuncGeneric,MethodTotal,MethodWithGenericReceiver,StructTotal,StructGeneric,StructGenericNonTrivialBound,StructAsTypeBound,TypeDecl,GenericTypeDecl,GenericTypeSet")

		for _, file := range files {
			counts, err := analyzeFile(file)
			if err != nil {
				log.Println("Error:", err)
			} else {
				aggregateCounters(&countersForProject, counts)
			}
		}

		// Ausgabe für lokales Projekt
		projectName := "local/" + filepath.Base(config.LocalProject)
		printCSVRow(projectName, countersForProject)

		// Gesamt-Statistik
		printCountersSummary(countersForProject, "Counter for local project")

		return
	}

	// === GITHUB MODUS (wie bisher) ===
	entries, _ := utils.GetOwnerAndRepo(config.CSVPath)

	// CSV-Header anpassen
	fmt.Println("Repository,FuncTotal,FuncGeneric,MethodTotal,MethodWithGenericReceiver,StructTotal,StructGeneric,StructGenericNonTrivialBound,StructAsTypeBound,TypeDecl,GenericTypeDecl,GenericTypeSet")

	for _, repository := range entries {
		files, err := utils.FetchGoFilesList(repository[0], repository[1], config.Token)
		if err != nil {
			log.Println(err)
		} else {
			countersForEntireRepo := GenericCounters{}

			for _, file := range files {
				counts, err := analyzeFile(file)
				if err != nil {
					log.Println("Error:", err)
				} else {
					aggregateCounters(&countersForEntireRepo, counts)
				}
			}

			// Aggregation auf Repository-Ebene
			if countersForEntireRepo.FuncGeneric > 0 {
				counterOverEveryRepository.FuncGeneric++
			}
			if countersForEntireRepo.MethodWithGenericReceiver > 0 {
				counterOverEveryRepository.MethodWithGenericReceiver++
			}
			if countersForEntireRepo.GenericTypeDecl > 0 {
				counterOverEveryRepository.GenericTypeDecl++
			}
			if countersForEntireRepo.GenericTypeSet > 0 {
				counterOverEveryRepository.GenericTypeSet++
			}
			if countersForEntireRepo.StructGeneric > 0 {
				counterOverEveryRepository.StructGeneric++
			}
			if countersForEntireRepo.StructGenericBound > 0 {
				counterOverEveryRepository.StructGenericBound++
			}

			log.Printf("Finished repository: %s/%s", repository[0], repository[1])

			// CSV-Ausgabe pro Repo
			repoName := repository[0] + "/" + repository[1]
			printCSVRow(repoName, countersForEntireRepo)
		}
	}

	// Gesamt-Statistik am Ende
	printCountersSummary(counterOverEveryRepository, "Counter over every Repository")
}
