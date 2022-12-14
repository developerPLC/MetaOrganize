package main

import (
	"bufio"
	"encoding/base64"
	"encoding/csv"
	"encoding/json"
	"flag"
	"fmt"
	"io/fs"
	"io/ioutil"
	"log"
	"math"
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
	Rarity     float64
}

type MainCounts struct {
	CountObjs []CountStruct
}

var dir string
var imageDir string = ""

func PrintUsage() {
	fmt.Printf("[ Usage ]\n")
	fmt.Printf("\t MetaOrganize -metadata { directory of JSON metadata }\n")
	fmt.Printf("\t MetaOrganize -metadata { directory of JSON metadata } -images { directory of images for HTML output }\n")
	os.Exit(1)
}

func main() {
	fmt.Printf("[ MetaOrganize by PLC ]\n")
	fmt.Printf("[ https://github.com/developerPLC/MetaOrganize ]\n")
	flag.StringVar(&dir, "metadata", "", "Directory of metadata ( ex example/metadata )")
	flag.StringVar(&imageDir, "images", "", "Directory of images ( ex example/images )")
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
	totalTokens := [][]string{}
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

		totalTokens = append(totalTokens, newRec)

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

	// Sort by Count
	sort.Slice(CountMap.CountObjs, func(i, j int) bool {
		return CountMap.CountObjs[i].Count < CountMap.CountObjs[j].Count
	})

	// Sort by Trait Type
	sort.Slice(CountMap.CountObjs, func(i, j int) bool {
		return CountMap.CountObjs[i].TraitType < CountMap.CountObjs[j].TraitType
	})

	// Spacing
	for x := 0; x < 3; x++ {
		records = append(records, []string{})
	}

	records = append(records, []string{"Trait Type", "Trait Value", "Count", "Rarity", "Ids"})

	// Add Counts
	var countHtmlObj [][]string
	for i, v := range CountMap.CountObjs {
		var idString string = ""
		for _, id := range v.Ids {
			if idString == "" {
				idString = fmt.Sprintf("%s", id)
			} else {
				idString = fmt.Sprintf("%s,%s", idString, id)
			}
		}

		// calculate raritya
		//fmt.Printf("[ rarity ] %d - %d\n", len(v.Ids), len(totalTokens))
		rarity := (float64(len(v.Ids)) / float64(len(totalTokens))) * 1000
		rarity = math.Round(rarity) / 10

		CountMap.CountObjs[i].Rarity = rarity

		newMap := []string{
			v.TraitType,
			v.TraitValue,
			fmt.Sprintf("%d", v.Count),
			fmt.Sprintf("%0.2f%%", rarity),
			idString,
		}

		countHtmlObj = append(countHtmlObj, newMap)
		records = append(records, newMap)
	}

	var HtmlBody string = "<div class='countLine pt20'><div><b>Trait Type</b></div><div><b>Trait Value</b></div><div><b>Count</b></div><div><b>Rarity</b></div></div>"
	// Start Adding Counter Objects to Body
	for _, obj := range CountMap.CountObjs {
		newCount := fmt.Sprintf("<div class='countLine'><div>%s</div><div>%s</div><div>%d</div><div>%0.2f %%</div></div>", obj.TraitType, obj.TraitValue, obj.Count, obj.Rarity)
		HtmlBody = fmt.Sprintf("%s\n%s", HtmlBody, newCount)
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

	// Create image name array
	imageFiles, _ := ioutil.ReadDir(imageDir)

	for _, record := range records {
		// Check if token record
		tokenId, isRec := IsTokenRecord(record)

		if isRec {
			fmt.Printf("[ record ] %v\n", record)
			var recordData string = ""

			// Check if image flag is null & grab image of token
			if imageDir != "" {
				// determine images & if extension
				tokenIdStr := fmt.Sprintf("%d", tokenId)
				imageToOpenFn, imageToOpenExt := GetImageFileName(&imageFiles, tokenIdStr)

				var imageToOpen string
				if imageToOpenExt == "" {
					// No Extension
					imageToOpen = fmt.Sprintf("%s/%s", imageDir, imageToOpenFn)
				} else {
					// Extension
					imageToOpen = fmt.Sprintf("%s/%s.%s", imageDir, imageToOpenFn, imageToOpenExt)
				}

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

				// If Image Extension
				if imageToOpenExt != "" {
					recordData = fmt.Sprintf("<div><img class='square' src='data:image/%s;base64,%s' /></div>", imageToOpenExt, encoded)
				} else {
					recordData = fmt.Sprintf("<div><img class='square' src='data:image/png;base64,%s' /></div>", encoded)
				}
			}

			// Add Record or li
			addedUl := false
			for x := 0; x < len(record); x++ {
				if x < 4 {
					recordData = fmt.Sprintf("%s<div>%s</div>", recordData, record[x])
				} else {
					if !addedUl {
						recordData = fmt.Sprintf("%s<ul><li>%s</li>", recordData, record[x])
						addedUl = true
					} else {
						if x == len(record) {
							recordData = fmt.Sprintf("%s<li>%s</li></ul>", recordData, record[x])
						} else {
							recordData = fmt.Sprintf("%s<li>%s</li>", recordData, record[x])
						}
					}
				}
			}

			// Add to HTML Template
			ContentToAdd := `<div class='flexRow'>` + recordData + `</div>`
			HtmlBody = fmt.Sprintf("%s\n%s", HtmlBody, ContentToAdd)
		}

		if err := w.Write(record); err != nil {
			log.Fatalln("error writing record to file", err)
		}
	}

	htmlStr := strings.Replace(GenHTMLTemplate(), ReplacementString, HtmlBody, 1)

	fmt.Printf("[ saving output.html ]\n")
	err = ioutil.WriteFile("output.html", []byte(htmlStr), 0755)
	if err != nil {
		log.Fatalf("[ error ] unable to write file output.html\n")
	}
}

// Reading CSV is token record
func IsTokenRecord(rec []string) (int64, bool) {
	var parsed int64
	if len(rec) > 0 {
		parsed, err := strconv.ParseInt(rec[0], 10, 32)
		if err == nil {
			return parsed, true
		}
	}
	return parsed, false
}

// Get Filename & Extension of image
func GetImageFileName(images *[]fs.FileInfo, id string) (string, string) {
	ImageNameRegex, _ := regexp.Compile(fmt.Sprintf("^%s((?:\\.gif|\\.svg|\\.png|))", id))
	for _, file := range *images {
		tokenId := ImageNameRegex.FindStringSubmatch(file.Name())
		// Extension
		if len(tokenId) > 0 {
			if tokenId[1] != "" {
				extOnly := strings.ReplaceAll(tokenId[1], ".", "")
				return id, extOnly
			} else {
				return id, ""
			}
		}
	}
	return "", ""
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
				body, html {
					font-family: sans-serif;
				}

				.square {
					max-width: 200px;
				}

				.container {
					display: flex;
					flex-direction: column;
					align-items: space-around;
				}

				.pt20 {
					padding-top: 20px;
				}

				.flexRow {
					padding: 10px;
					display: flex;
					justify-content: space-around;
					align-items: center;
					margin: 15px;
					box-shadow: 1px 2px 2px 2px #0003;
				}

				.countLine {
					display: flex;
					font-size: 0.8em;
					justify-content: space-around;
					align-items: center;
					padding: 2px;
				}

				ul li {
					padding: 10px;
					border-bottom: 1px solid #0003;
				}

			</style>
			<title>MetaOrganize by PLC</title>
		</head>
		<body>
			<div class="container">
				<h1 style="text-align: center;">MetaOrganize by PLC</h1>
				<div style="text-align: center;"><a href="https://github.com/developerPLC/MetaOrganize" target="_blank">MetaOrganize GitHub</a></div>
				{ GeneratedBody } 
			</div>
		</body>
</html>
	`)
}
