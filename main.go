package main

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"
)

type OptionBank struct {
	Name    string   `json:"name"`
	Options []Option `json:"options"`
}

type Option struct {
	Label string `json:"label"`
	Value string `json:"value"`
}

func main() {
	var token = flag.String("token", "", "API token for Fountain")
	var filePath = flag.String("file", "", "File path for csv to read option banks from")
	var appURL = flag.String("url", "https://api.fountain.com/v2/", "Base API URL. Defaults to https://api.fountain.com/v2/")
	flag.Parse()
	if len(*token) == 0 || len(*filePath) == 0 {
		println("Token and file path are required")
		os.Exit(1)
	}
	log.Print(*appURL)
	optionBanks, err := readCSV(*filePath)
	if err != nil {
		log.Printf("Unable to read options CSV: %v", err)
		return
	}

	client := http.Client{
		Timeout: 20 * time.Second,
	}
	saved, err := submitOptions(*appURL, *token, &client, optionBanks)
	if len(saved) > 0 {
		log.Println("Saved:")
		for _, name := range saved {
			log.Println(name)
		}
	}
	if err != nil {
		log.Println("Failed to create an option bank")
		log.Print(err)
	}
}

func readCSV(filePath string) ([]OptionBank, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	reader := csv.NewReader(file)
	reader.Read() // discard header row
	banks := make([]OptionBank, 0)
	currentBank := OptionBank{}
	for {
		line, err := reader.Read()
		if err != nil {
			break
		}
		if len(line[0]) > 0 {
			if len(currentBank.Name) > 0 {
				banks = append(banks, currentBank)
			}
			currentBank = OptionBank{Name: line[0], Options: make([]Option, 0)}
		}
		currentBank.Options = append(currentBank.Options, Option{Label: line[1], Value: line[2]})
	}
	banks = append(banks, currentBank)

	return banks, nil
}

func submitOptions(url string, token string, client *http.Client, banks []OptionBank) ([]string, error) {
	buffer := new(bytes.Buffer)
	successful := make([]string, 0)
	for _, bank := range banks {
		buffer.Reset()
		json.NewEncoder(buffer).Encode(&bank)
		req, err := http.NewRequest("POST", url, buffer)
		if err != nil {
			return successful, err
		}
		req.Header.Add("X-ACCESS-TOKEN", token)
		req.Header.Add("content-type", "application/json")
		resp, err := client.Do(req)
		if err != nil {
			return successful, err
		}
		if resp.StatusCode > 299 {
			data, _ := io.ReadAll(resp.Body)
			log.Print("CREATE failed")
			log.Print(string(data))
			return successful, fmt.Errorf("failed to create option bank %s: %d", bank.Name, resp.StatusCode)
		}
		successful = append(successful, bank.Name)
	}
	return successful, nil
}
