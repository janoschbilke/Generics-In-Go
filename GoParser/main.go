package main

import (
	"fmt"
	"log"
)

func main() {
	config, err := SetupEnvironment()
	if err != nil {
		log.Fatalf("Failed to set up environment: %v", err)
	}

	entries, _ := getOwnerAndRepo(config.CSVPath)

	counterOverEveryRepository := GenericCounters{}

	// CSV-Header anpassen
	fmt.Println("Repository,FuncTotal,FuncGeneric,MethodTotal,MethodWithGenericReceiver,StructTotal,StructGeneric,StructGenericNonTrivialBound,TypeDecl,GenericTypeDecl,GenericTypeSet")

	for _, repository := range entries {
		files, err := fetchGoFilesList(repository[0], repository[1], config.Token)
		if err != nil {
			log.Println(err)
		} else {
			countersForEntireRepo := GenericCounters{}

			for _, file := range files {
				counts, err := analyzeFile(file)
				if err != nil {
					log.Println("Error:", err)
				} else {
					// alle Felder aufsummieren
					countersForEntireRepo.FuncTotal += counts.FuncTotal
					countersForEntireRepo.FuncGeneric += counts.FuncGeneric
					countersForEntireRepo.MethodTotal += counts.MethodTotal
					countersForEntireRepo.MethodWithGenericReceiver += counts.MethodWithGenericReceiver
					countersForEntireRepo.StructTotal += counts.StructTotal
					countersForEntireRepo.StructGeneric += counts.StructGeneric
					countersForEntireRepo.StructGenericBound += counts.StructGenericBound
					countersForEntireRepo.TypeDecl += counts.TypeDecl
					countersForEntireRepo.GenericTypeDecl += counts.GenericTypeDecl
					countersForEntireRepo.GenericTypeSet += counts.GenericTypeSet
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
			fmt.Printf("%s/%s,%d,%d,%d,%d,%d,%d,%d,%d,%d,%d\n",
				repository[0], repository[1],
				countersForEntireRepo.FuncTotal,
				countersForEntireRepo.FuncGeneric,
				countersForEntireRepo.MethodTotal,
				countersForEntireRepo.MethodWithGenericReceiver,
				countersForEntireRepo.StructTotal,
				countersForEntireRepo.StructGeneric,
				countersForEntireRepo.StructGenericBound,
				countersForEntireRepo.TypeDecl,
				countersForEntireRepo.GenericTypeDecl,
				countersForEntireRepo.GenericTypeSet,
			)
		}
	}

	// Gesamt-Statistik am Ende
	fmt.Println()
	fmt.Println("Counter over every Repository:")
	fmt.Printf("FuncGeneric: %v\n", counterOverEveryRepository.FuncGeneric)
	fmt.Printf("MethodWithGenericReceiver: %v\n", counterOverEveryRepository.MethodWithGenericReceiver)
	fmt.Printf("StructGeneric: %v\n", counterOverEveryRepository.StructGeneric)
	fmt.Printf("StructGenericNonTrivialBound: %v\n", counterOverEveryRepository.StructGenericBound)
	fmt.Printf("GenericTypeDecl: %v\n", counterOverEveryRepository.GenericTypeDecl)
	fmt.Printf("GenericTypeSet: %v\n", counterOverEveryRepository.GenericTypeSet)
}
