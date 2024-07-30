package main

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"kelurahanMapper/db"
	"net/http"
	"os"
	"slices"
	"strconv"
	"strings"

	"github.com/schollz/progressbar/v3"
)

const (
	overpassURL = "https://overpass-api.de/api/interpreter"
)

type (
	Node struct {
		Lat, Lon float64
	}
	kelurahanDb struct {
		id   int
		name string
	}
)
type OverpassResponse struct {
	Version   float64           `json:"version"`
	Generator string            `json:"generator"`
	Osm3s     Osm3sData         `json:"osm3s"`
	Elements  []OverpassElement `json:"elements"`
}

type Osm3sData struct {
	TimestampOsmBase   string `json:"timestamp_osm_base"`
	TimestampAreasBase string `json:"timestamp_areas_base"`
	Copyright          string `json:"copyright"`
}

type OverpassElement Relation // Can hold various element types

type Relation struct {
	Type   string            `json:"type"`
	ID     int               `json:"id"`
	Bounds Bounds            `json:"bounds"`
	Tags   map[string]string `json:"tags"`
}

type Bounds struct {
	Minlat float64 `json:"minlat"`
	Minlon float64 `json:"minlon"`
	Maxlat float64 `json:"maxlat"`
	Maxlon float64 `json:"maxlon"`
}

func main() {

	db, err := db.ConnectDb(context.Background())
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	resultKelurahan, err := db.Query(context.Background(), `
	SELECT 
		* 
	FROM 
	    daftar_kelurahan`)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	defer resultKelurahan.Close()
	resultRows := make([]kelurahanDb, 0)
	for resultKelurahan.Next() {
		row := kelurahanDb{}
		err = resultKelurahan.Scan(&row.id, &row.name)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			return
		}

		resultRows = append(resultRows, row)
	}

	query := `
		[out:json];
		is_in(%f,%f)->.a;
		relation(pivot.a)[admin_level=7];
		out tags bb;
    `

	csvFile, err := os.Open("public places.csv")
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	csvReader := csv.NewReader(csvFile)
	defer csvFile.Close()

	records, err := csvReader.ReadAll()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	bar1 := progressbar.Default(int64(len(records)))
	modifiedRecords := records
	for i, record := range records {
		bar1.Add(1)
		if i == 0 {
			continue
		}
		lat, err := strconv.ParseFloat(record[13], 64)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			return
		}
		lon, err := strconv.ParseFloat(record[14], 64)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			return
		}

		KelurahanQuery := fmt.Sprintf(query, float64(lat), float64(lon))
		result, err := fetchOverpassData(KelurahanQuery)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			return
		}

		for _, element := range result.Elements {
			if element.Tags["admin_level"] == "7" {
				kelurhanIndex := slices.IndexFunc(resultRows, func(kelurahan kelurahanDb) bool {
					return strings.ReplaceAll(strings.ToLower(kelurahan.name), " ", "") == strings.ReplaceAll(strings.ToLower(element.Tags["name"]), " ", "")
				})
				if kelurhanIndex == -1 {
					continue
				}
				modifiedRecords[i] = append(record, strconv.Itoa(resultRows[kelurhanIndex].id))
				modifiedRecords[i] = append(modifiedRecords[i], resultRows[kelurhanIndex].name)
				break
			}
		}
	}

	fmt.Println()
	fmt.Println()
	fmt.Println()
	fmt.Println("WRITING")

	csvFile, err = os.Create("public places_mapped.csv")
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	csvWriter := csv.NewWriter(csvFile)
	err = csvWriter.WriteAll(modifiedRecords)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

}

func fetchOverpassData(query string) (response OverpassResponse, err error) {
	resp, err := sendOverpassRequest(query)
	if err != nil {
		return response, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return response, fmt.Errorf("failed to read response body: %w", err)
	}

	if err := json.Unmarshal(body, &response); err != nil {
		return response, fmt.Errorf("failed to unmarshal JSON: %w", err)
	}

	return response, nil
}

func sendOverpassRequest(query string) (*http.Response, error) {
	queryStr := "data=" + query
	req, err := http.NewRequest("POST", overpassURL, bytes.NewBufferString(queryStr))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{}
	return client.Do(req)
}
