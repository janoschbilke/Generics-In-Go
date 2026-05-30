package utils

import (
	"GoParser/model"
	"encoding/csv"
	"fmt"
	"os"
	"sort"
	"strings"
)

// GetOwnerAndRepo liest eine CSV-Datei ein und gibt für jede Zeile owner und repo zurück
func GetOwnerAndRepo(filename string) ([][2]string, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		return nil, err
	}

	var result [][2]string
	for i, record := range records {
		// Überspringe die Kopfzeile
		if i == 0 {
			continue
		}
		if len(record) < 2 {
			continue
		}

		// Repository im Format "github.com/owner/repo"
		parts := strings.Split(record[1], "/")
		if len(parts) < 3 {
			continue
		}
		owner := parts[1]
		repo := parts[2]

		result = append(result, [2]string{owner, repo})
	}

	return result, nil
}

func PrintCSVRow(name string, counters model.GenericCounters) {
	fmt.Printf("%s,%d,%d,%d,%d,%d,%d,%d,%d,%d,%d,%d,%d,%d,%d,%d,%d,%d,%d\n",
		name,
		counters.FuncTotal,
		counters.FuncGeneric,
		counters.MethodTotal,
		counters.MethodWithGenericReceiver,
		counters.MethodWithGenericReceiverTrivialTypeBound,
		counters.MethodWithGenericReceiverNonTrivialTypeBound,
		counters.StructTotal,
		counters.StructGeneric,
		counters.StructGenericBound,
		counters.StructAsTypeBound,
		counters.TypeDecl,
		counters.GenericTypeDecl,
		counters.GenericTypeSet,
		counters.GenericFuncInstantiationExplicit,
		counters.GenericFuncInstantiationInferred,
		counters.GenericTypeInstantiationExplicit,
		counters.GenericTypeInstantiationInferred,
		counters.GenericMethodInstantiationInferred,
	)
}

func PrintCSVHeader() {
	fmt.Println("Repository,FuncTotal,FuncGeneric,MethodTotal,MethodWithGenericReceiver,MethodWithGenericReceiverTrivialTypeBound,MethodWithGenericReceiverNonTrivialTypeBound,StructTotal,StructGeneric,StructGenericNonTrivialBound,StructAsTypeBound,TypeDecl,GenericTypeDecl,GenericTypeSet,GenericFuncInstantiationExplicit,GenericFuncInstantiationInferred,GenericTypeInstantiationExplicit,GenericTypeInstantiationInferred,GenericMethodInstantiationInferred")
}

// ComputeCrossRepoAggregation counts how many repositories have at least one occurrence
// of each generic feature. This is the GitHub-mode summary statistic.
func ComputeCrossRepoAggregation(results []model.GenericCounters) model.GenericCounters {
	summary := model.GenericCounters{}
	for _, r := range results {
		if r.FuncGeneric > 0 {
			summary.FuncGeneric++
		}
		if r.MethodWithGenericReceiver > 0 {
			summary.MethodWithGenericReceiver++
		}
		if r.GenericTypeDecl > 0 {
			summary.GenericTypeDecl++
		}
		if r.GenericTypeSet > 0 {
			summary.GenericTypeSet++
		}
		if r.StructGeneric > 0 {
			summary.StructGeneric++
		}
		if r.StructGenericBound > 0 {
			summary.StructGenericBound++
		}
	}
	return summary
}

func PrintInstantiationSummary(projectName string, data model.InstantiationData) {
	if len(data) == 0 {
		return
	}

	var structKeys, funcKeys, methodKeys []string
	for key, entries := range data {
		for _, entry := range entries {
			switch entry.Kind {
			case model.KindStruct:
				structKeys = append(structKeys, key)
			case model.KindFunction:
				funcKeys = append(funcKeys, key)
			case model.KindMethod:
				methodKeys = append(methodKeys, key)
			}
			break
		}
	}
	sort.Strings(structKeys)
	sort.Strings(funcKeys)
	sort.Strings(methodKeys)

	fmt.Printf("\nInstantiation diversity for %s:\n", projectName)

	if len(structKeys) > 0 {
		fmt.Println("  [Structs]")
		for _, key := range structKeys {
			printInstantiationLine(key, data)
		}
	}

	if len(funcKeys) > 0 {
		fmt.Println("  [Functions]")
		for _, key := range funcKeys {
			printInstantiationLine(key, data)
		}
	}

	if len(methodKeys) > 0 {
		fmt.Println("  [Methods on generic types]")
		for _, key := range methodKeys {
			printInstantiationLine(key, data)
		}
	}
}

func printInstantiationLine(key string, data model.InstantiationData) {
	concrete := data.ConcreteEntries(key)
	parametric := data.ParametricEntries(key)

	displayName := key
	for _, entry := range data[key] {
		displayName = entry.Name
		break
	}

	parts := []string{}
	if len(concrete) > 0 {
		parts = append(parts, fmt.Sprintf("%d concrete [%s]", len(concrete), strings.Join(concrete, ", ")))
	}
	if len(parametric) > 0 {
		parts = append(parts, fmt.Sprintf("%d parametric [%s]", len(parametric), strings.Join(parametric, ", ")))
	}
	fmt.Printf("    %s: %s\n", displayName, strings.Join(parts, " + "))
}

func PrintCountersSummary(counters model.GenericCounters, title string) {
	fmt.Println()
	fmt.Printf("%s:\n", title)
	fmt.Printf("FuncGeneric: %v\n", counters.FuncGeneric)
	fmt.Printf("MethodWithGenericReceiver: %v\n", counters.MethodWithGenericReceiver)
	fmt.Printf("MethodWithGenericReceiverTrivialTypeBound: %v\n", counters.MethodWithGenericReceiverTrivialTypeBound)
	fmt.Printf("MethodWithGenericReceiverNonTrivialTypeBound: %v\n", counters.MethodWithGenericReceiverNonTrivialTypeBound)
	fmt.Printf("StructGeneric: %v\n", counters.StructGeneric)
	fmt.Printf("StructGenericNonTrivialBound: %v\n", counters.StructGenericBound)
	fmt.Printf("StructAsTypeBound: %v\n", counters.StructAsTypeBound)
	fmt.Printf("GenericTypeDecl: %v\n", counters.GenericTypeDecl)
	fmt.Printf("GenericTypeSet: %v\n", counters.GenericTypeSet)

	if counters.GenericFuncInstantiationExplicit > 0 || counters.GenericFuncInstantiationInferred > 0 ||
		counters.GenericTypeInstantiationExplicit > 0 || counters.GenericTypeInstantiationInferred > 0 ||
		counters.GenericMethodInstantiationInferred > 0 {
		fmt.Println("\nGeneric Instantiations:")
		fmt.Printf("  GenericFuncInstantiationExplicit:  %v\n", counters.GenericFuncInstantiationExplicit)
		fmt.Printf("  GenericFuncInstantiationInferred:  %v\n", counters.GenericFuncInstantiationInferred)
		fmt.Printf("  GenericTypeInstantiationExplicit:  %v\n", counters.GenericTypeInstantiationExplicit)
		fmt.Printf("  GenericTypeInstantiationInferred:  %v\n", counters.GenericTypeInstantiationInferred)
		fmt.Printf("  GenericMethodInstantiationInferred: %v\n", counters.GenericMethodInstantiationInferred)
	}
}
