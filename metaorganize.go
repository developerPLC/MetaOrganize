package main

import (
	"encoding/csv"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"regexp"
	"sort"
)

type Trait struct {
	TraitType  interface{} `json:"trait_type"`
	TraitValue interface{} `json:"value"`
}

type MetaData struct {
	Name        string  `json:"name"`
	ExternalUrl string  `json:"external_url"`
	Image       string  `json:"image"`
	Attributes  []Trait `json:"attributes"`
	Description string  `json:"description"`
}

type CountStruct struct {
	TraitType  string
	TraitValue string
	Count      int
	Ids        []string
}

type MainCounts struct {
	CountObjs []CountStruct
}

var dir string

func main() {
	fmt.Printf("[ meta organize by PLC.eth ]\n")
	flag.StringVar(&dir, "dir", "", "Directory of metadata")
	flag.Parse()

	if dir == "" {
		fmt.Printf("[ usage ] metaorganize -dir { directory of metadata }")
		os.Exit(1)
	}

	// Get Length of Metadata Dir
	files, err := ioutil.ReadDir(dir)
	if err != nil {
		log.Fatal(err)
	}

	records := [][]string{
		{
			"id",
			"name",
			"image",
			"external_url",
		},
	}

	CountMap := MainCounts{}

	idRegex, _ := regexp.Compile("([0-9]+)")

	// Loop Each Metadata file
	for _, file := range files {
		if file.IsDir() {
			continue
		}

		fileToOpen := fmt.Sprintf("%s/%s", dir, file.Name())
		fmt.Printf("%s\n", fileToOpen)

		// Open File
		curDataFile, err := os.Open(fileToOpen)
		if err != nil {
			log.Printf("[ error ] could not open file.\t%s\n", fileToOpen)
			continue
		}
		defer curDataFile.Close()

		// Read all metadata
		curDataFileBytes, err := ioutil.ReadAll(curDataFile)
		if err != nil {
			log.Fatalf("[ error ] could not read metadata file %s\n", fileToOpen)
		}

		// create new metadata object
		var md MetaData
		err = json.Unmarshal(curDataFileBytes, &md)
		if err != nil {
			log.Fatalf("[ incorrect metadata ] %+v\n", err)
		}

		// Get token id from name
		matchedId := idRegex.FindString(fileToOpen)
		if matchedId == "" {
			fmt.Printf("[ unable to determine token id ] %s\n", fileToOpen)
			continue
		}

		// add record
		newRec := []string{
			matchedId,
			md.Name,
			md.Image,
			md.ExternalUrl,
		}

		for _, attr := range md.Attributes {
			traitTypeString := fmt.Sprintf("%v", attr.TraitType)
			traitValString := fmt.Sprintf("%v", attr.TraitValue)

			newRec = append(newRec, fmt.Sprintf("%s - %v", traitTypeString, traitValString))

			if CountContains(CountMap.CountObjs, traitTypeString, traitValString) {
				// Already there
				CountMap.UpCount(traitTypeString, traitValString)
				CountMap.AddId(traitTypeString, traitValString, matchedId)
			} else {
				// Add to array
				newObj := CountStruct{
					TraitType:  traitTypeString,
					TraitValue: traitValString,
					Count:      1,
					Ids:        []string{matchedId},
				}

				CountMap.CountObjs = append(CountMap.CountObjs, newObj)
			}
		}

		records = append(records, newRec)
	}

	// Sort
	sort.Slice(CountMap.CountObjs, func(i, j int) bool {
		return CountMap.CountObjs[i].Count < CountMap.CountObjs[j].Count
	})

	// Spacing
	for x := 0; x < 3; x++ {
		records = append(records, []string{})
	}

	// Add Counts
	for _, v := range CountMap.CountObjs {
		var idString string = ""
		for _, id := range v.Ids {
			if idString == "" {
				idString = fmt.Sprintf("%s", id)
			} else {
				idString = fmt.Sprintf("%s,%s", idString, id)
			}
		}

		newMap := []string{
			v.TraitType,
			v.TraitValue,
			fmt.Sprintf("%d", v.Count),
			idString,
		}
		records = append(records, newMap)
	}

	// SAVE CSV RECORDS
	fmt.Printf("[ saving output.csv ]\n")
	f, err := os.Create("output.csv")
	if err != nil {
		log.Fatalln("failed to open output for saving", err)
	}
	defer f.Close()

	// Write CSV
	w := csv.NewWriter(f)
	defer w.Flush()
	for _, record := range records {
		if err := w.Write(record); err != nil {
			log.Fatalln("error writing record to file", err)
		}
	}
}

// If Count Object Contains
func CountContains(s []CountStruct, traitType string, traitValue string) bool {
	for _, a := range s {
		if a.TraitType == traitType && a.TraitValue == traitValue {
			return true
		}
	}
	return false
}

// Add to Count
func (ms *MainCounts) UpCount(traitType string, traitValue string) {
	for i, a := range ms.CountObjs {
		if a.TraitType == traitType && a.TraitValue == traitValue {
			ms.CountObjs[i].Count = ms.CountObjs[i].Count + 1
		}
	}
}

// Add ID To Array
func (ms *MainCounts) AddId(traitType string, traitValue string, id string) {
	for i, a := range ms.CountObjs {
		if a.TraitType == traitType && a.TraitValue == traitValue {
			ms.CountObjs[i].Ids = append(ms.CountObjs[i].Ids, id)
		}
	}
}
