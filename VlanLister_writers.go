package main

/*
#### ##     ## ########   #######  ########  ########  ######
 ##  ###   ### ##     ## ##     ## ##     ##    ##    ##    ##
 ##  #### #### ##     ## ##     ## ##     ##    ##    ##
 ##  ## ### ## ########  ##     ## ########     ##     ######
 ##  ##     ## ##        ##     ## ##   ##      ##          ##
 ##  ##     ## ##        ##     ## ##    ##     ##    ##    ##
#### ##     ## ##         #######  ##     ##    ##     ######
*/

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"

	excelize "github.com/360EntSecGroup-Skylar/excelize/v2"
)

/*
##     ##    ###    ########   ######
##     ##   ## ##   ##     ## ##    ##
##     ##  ##   ##  ##     ## ##
##     ## ##     ## ########   ######
 ##   ##  ######### ##   ##         ##
  ## ##   ##     ## ##    ##  ##    ##
   ###    ##     ## ##     ##  ######
*/

var (
	// File types that are valid for writing
	validFiletypes = [...]string{"csv", "json", "stdout", "xlsx"}
)

/*
######## ##     ## ##    ##  ######   ######
##       ##     ## ###   ## ##    ## ##    ##
##       ##     ## ####  ## ##       ##
######   ##     ## ## ## ## ##        ######
##       ##     ## ##  #### ##             ##
##       ##     ## ##   ### ##    ## ##    ##
##        #######  ##    ##  ######   ######
*/

// Decides which actual writeResults* function shall be used based on filename pre- or suffix
func writeResults(filename string, resultsNew devicesWrapper) (uint, error) {
	// Prefix checking
	for _, filetype := range validFiletypes {
		prefix := fmt.Sprintf("%s:", filetype)
		if strings.HasPrefix(filename, prefix) {
			filename = strings.TrimPrefix(filename, prefix)
			switch filetype {
			case "csv":
				return writeResultsCSV(filename, resultsNew)
			case "json":
				return writeResultsJSON(filename, resultsNew)
			case "stdout":
				return writeResultsStdout(filename, resultsNew)
			case "xlsx":
				return writeResultsXLSX(filename, resultsNew)
			}
		}
	}

	// Suffix checking
	for _, filetype := range validFiletypes {
		suffix := fmt.Sprintf(".%s", filetype)
		if strings.HasSuffix(filename, suffix) {
			switch filetype {
			case "csv":
				return writeResultsCSV(filename, resultsNew)
			case "json":
				return writeResultsJSON(filename, resultsNew)
			case "stdout":
				return writeResultsStdout(filename, resultsNew)
			case "xlsx":
				return writeResultsXLSX(filename, resultsNew)
			}
		}
	}

	return 0, fmt.Errorf("Could not determine file type for <%s>", filename)
}

// Writes the results to outfile in CSV format
func writeResultsCSV(filename string, results devicesWrapper) (uint, error) {
	var rowsWritten uint = 0

	csvData, csvError := results.ToCSV()
	if csvError != nil {
		return rowsWritten, fmt.Errorf("Could not convert data to CSV: %s", csvError)
	}

	fileHandle, fileErr := os.Create(filename)
	if fileErr != nil {
		return rowsWritten, fmt.Errorf("Could not create outfile: %s", fileErr)
	}
	fileWriter := bufio.NewWriter(fileHandle)
	for _, line := range strings.Split(csvData, "\n") {
		_, writeErr := fileWriter.WriteString(fmt.Sprintf("%s\n", line))
		if writeErr != nil {
			return rowsWritten, fmt.Errorf("Could not write to outfile: %s", writeErr)
		}
		flushErr := fileWriter.Flush()
		if flushErr != nil {
			stdErr.Printf("Could not flush file buffer: %s\n", flushErr)
		}
		rowsWritten++
	}
	syncErr := fileHandle.Sync()
	if syncErr != nil {
		stdErr.Printf("Could not sync file handle: %s\n", syncErr)
	}
	fhErr := fileHandle.Close()
	if fhErr != nil {
		stdErr.Printf("Could not close file handle: %s\n", fhErr)
	}

	return rowsWritten, nil
}

// Writes the results to outfile in JSON format
func writeResultsJSON(filename string, results devicesWrapper) (uint, error) {
	var rowsWritten uint = 0

	jsonData, jsonErr := results.ToJSON()
	if jsonErr != nil {
		return rowsWritten, fmt.Errorf("Could not encode JSON: %s", jsonErr)
	}

	fileHandle, fileErr := os.Create(filename)
	if fileErr != nil {
		return rowsWritten, fmt.Errorf("Could not create outfile: %s", fileErr)
	}
	fileWriter := bufio.NewWriter(fileHandle)
	for _, line := range strings.Split(jsonData, "\n") {
		_, writeErr := fileWriter.WriteString(fmt.Sprintf("%s\n", line))
		if writeErr != nil {
			return rowsWritten, fmt.Errorf("Could not write to outfile: %s", writeErr)
		}
		flushErr := fileWriter.Flush()
		if flushErr != nil {
			stdErr.Printf("Could not flush file buffer: %s\n", flushErr)
		}
		rowsWritten++
	}
	syncErr := fileHandle.Sync()
	if syncErr != nil {
		stdErr.Printf("Could not sync file handle: %s\n", syncErr)
	}
	fhErr := fileHandle.Close()
	if fhErr != nil {
		stdErr.Printf("Could not close file handle: %s\n", fhErr)
	}

	return rowsWritten, nil
}

// Writes the results to stdout in CSV format
func writeResultsStdout(filename string, results devicesWrapper) (uint, error) {
	var rowsWritten uint = 0

	csvData, csvError := results.ToCSV()
	if csvError != nil {
		return rowsWritten, fmt.Errorf("Could not convert data to CSV: %s", csvError)
	}

	for _, line := range strings.Split(csvData, "\n") {
		fmt.Printf("%s\n", line)
		rowsWritten++
	}

	return rowsWritten, nil
}

// Writes the results to outfile in XLSX format
func writeResultsXLSX(filename string, results devicesWrapper) (uint, error) {
	var rowsWritten uint = 0
	var colIndex int = 1
	var rowIndex int = 1
	var colors map[int]map[int]string
	var cellStyles map[int]map[int]int
	var devCount int = 0
	var rowCount int = 0
	var devStyleID int = 0
	var rowStyleID int = 0

	xlsx := excelize.NewFile()

	if !config.NoColor {
		colors = make(map[int]map[int]string)
		// Grey
		colors[0] = make(map[int]string)
		colors[0][0] = "#F2F2F2"
		colors[0][1] = "#E6E6E6"
		// Yellow
		colors[1] = make(map[int]string)
		colors[1][0] = "#FFFFE6"
		colors[1][1] = "#FFFFCC"
		// Green
		colors[2] = make(map[int]string)
		colors[2][0] = "#E6FFE6"
		colors[2][1] = "#CCFFCC"
		// Turqoise
		colors[3] = make(map[int]string)
		colors[3][0] = "#E6FFFF"
		colors[3][1] = "#CCFFFF"
		// Blue
		colors[4] = make(map[int]string)
		colors[4][0] = "#E6E6FF"
		colors[4][1] = "#CCCCFF"
		// Purple
		colors[5] = make(map[int]string)
		colors[5][0] = "#FFE6FF"
		colors[5][1] = "#FFCCFF"
		// Red
		colors[6] = make(map[int]string)
		colors[6][0] = "#FFE6E6"
		colors[6][1] = "#FFCCCC"

		cellStyles = make(map[int]map[int]int)
		for baseColor := range colors {
			cellStyles[baseColor] = make(map[int]int)
			for rowColor := range colors[baseColor] {
				styleString := fmt.Sprintf(`{"fill":{"type":"pattern","color":["%s"],"pattern":1}}`, colors[baseColor][rowColor])
				styleFormat, styleFormatErr := xlsx.NewStyle(styleString)
				if styleFormatErr != nil {
					return rowsWritten, styleFormatErr
				}
				cellStyles[baseColor][rowColor] = styleFormat
			}
		}
	}

	for _, columnName := range csvColumns {
		position, positionErr := excelize.CoordinatesToCellName(colIndex, rowIndex)
		if positionErr != nil {
			return rowsWritten, positionErr
		}
		valueErr := xlsx.SetCellValue("Sheet1", position, columnName)
		if valueErr != nil {
			stdErr.Printf("Could not set value for %s: %s", position, valueErr)
		}
		colIndex++
	}
	rowsWritten++

	for _, dev := range results.Devices {
		csvRows, csvRowsErr := dev.ToCSVRows()
		if csvRowsErr != nil {
			stdErr.Printf("Could not convert device to CSV rows: %s", csvRowsErr)
			continue
		}
		for _, row := range csvRows {
			colIndex = 1
			rowIndex++
			if !config.NoColor {
				devStyleID = devCount % len(cellStyles)
				rowStyleID = rowCount % len(cellStyles[devStyleID])
			}
			for _, element := range strings.Split(row, `","`) {
				element = strings.Trim(element, `"`)
				position, positionErr := excelize.CoordinatesToCellName(colIndex, rowIndex)
				if positionErr != nil {
					return rowsWritten, positionErr
				}
				valueErr := xlsx.SetCellValue("Sheet1", position, element)
				if valueErr != nil {
					stdErr.Printf("Could not set value for %s: %s", position, valueErr)
				}
				if !config.NoColor {
					styleErr := xlsx.SetCellStyle("Sheet1", position, position, cellStyles[devStyleID][rowStyleID])
					if styleErr != nil {
						stdErr.Printf("Could not set style for cell %s: %s", position, styleErr)
					}
				}
				colIndex++
			}
			rowsWritten++
			rowCount++
		}
		devCount++
		rowCount = 0
	}

	xlsx.SetSheetName("Sheet1", time.Now().Format(time.RFC3339))

	if saveErr := xlsx.SaveAs(filename); saveErr != nil {
		return rowsWritten, saveErr
	}

	return rowsWritten, nil
}
