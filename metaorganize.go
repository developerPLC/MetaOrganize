package main

import (
	"bufio"
	"encoding/base64"
	"encoding/csv"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
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
var imageDir string = ""
var imageExt string = ""

func PrintUsage() {
	fmt.Printf("[ Usage ]\n")
	fmt.Printf("\t MetaOrganize -dir { directory of JSON metadata }\n")
	fmt.Printf("\t MetaOrganize -dir { directory of JSON metadata } -images { directory of images for HTML output }\n")
	os.Exit(1)
}

func main() {
	fmt.Printf("[ meta organize by PLC.eth ]\n")
	flag.StringVar(&dir, "dir", "", "Directory of metadata ( ex example/metadata )")
	flag.StringVar(&imageDir, "images", "", "Directory of images ( ex example/images )")
	flag.StringVar(&imageExt, "imageext", "", "Extension of Images (ex .png )")
	flag.Parse()

	if dir == "" {
		PrintUsage()
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

	// Sort files by token #
	sort.Slice(files, func(i, j int) bool {
		matchedIda := idRegex.FindString(files[i].Name())
		matchedIdb := idRegex.FindString(files[j].Name())
		a, err := strconv.ParseInt(matchedIda, 10, 32)
		if err != nil {
			log.Fatalf("[ error ] %+v\n", err)
		}
		b, err := strconv.ParseInt(matchedIdb, 10, 32)
		if err != nil {
			log.Fatalf("[ error ] %+v\n", err)
		}

		return a < b
	})

	// Loop Each Metadata file
	for _, file := range files {
		if file.IsDir() {
			continue
		}

		fileToOpen := fmt.Sprintf("%s/%s", dir, file.Name())

		// Get token id from name
		matchedId := idRegex.FindString(fileToOpen)
		if matchedId == "" {
			fmt.Printf("[ unable to determine token id ] %s\n", fileToOpen)
			continue
		}

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

	var HtmlBody string
	for _, record := range records {
		// Check if token record
		tokenId, isRec := IsTokenRecord(record)

		if isRec {
			fmt.Printf("[ record ] %v\n", record)

			var recordData string = ""

			// Check if image flag is null & grab image of token
			if imageDir != "" {
				imageToOpen := fmt.Sprintf("%s/%d%s", imageDir, tokenId, imageExt)

				// attempt to open files
				f, err := os.Open(imageToOpen)
				if err != nil {
					log.Printf("[ error opening image ] %s\n", imageToOpen)
					break
				}
				defer f.Close()

				// Read entire Image into byte slice.
				reader := bufio.NewReader(f)
				content, _ := ioutil.ReadAll(reader)
				encoded := base64.StdEncoding.EncodeToString(content)

				recordData = fmt.Sprintf("<div><img class='square' src='data:image/png;base64,%s' /></div>", encoded)
			}

			for x := 0; x < len(record); x++ {
				recordData = fmt.Sprintf("%s<div>%s</div>", recordData, record[x])
			}

			ContentToAdd := `<div class='flexRow'>` + recordData + `</div>`
			// Add to HTML Template
			HtmlBody = fmt.Sprintf("%s\n%s", HtmlBody, ContentToAdd)
		}

		if err := w.Write(record); err != nil {
			log.Fatalln("error writing record to file", err)
		}
	}

	htmlStr := strings.Replace(GenHTMLTemplate(), ReplacementString, HtmlBody, 1)
	ioutil.WriteFile("output.html", []byte(htmlStr), 0755)
}

func IsTokenRecord(rec []string) (int64, bool) {
	var parsed int64
	if len(rec) > 0 {
		parsed, err := strconv.ParseInt(rec[0], 10, 32)
		if err == nil {
			//fmt.Printf("[ true ] %+v\n", parsed)
			return parsed, true
		}
	}
	return parsed, false
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

const ReplacementString = `{ GeneratedBody }`

func GenHTMLTemplate() string {
	return (`
<html>
		<head>
			<style>
				.square {
					max-width: 200px;
				}

				.container {
					display: flex;
					flex-direction: column;
					align-items: space-around;
				}

				.flexRow {
					padding: 10px;
					display: flex;
					justify-content: space-around;
					align-items: center;
					border-bottom: 1px solid #000;
				}

			</style>
			<title>MetaOrganize by PLC.eth</title>
		</head>
		<body>
			<div class="container">
				{ GeneratedBody } 
			</div>
		</body>
</html>

	`)
}
