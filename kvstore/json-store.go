package kvstore

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
)

type JsonStore struct {
	filePath string
}

func NewJsonStore(filePath string) *JsonStore {
	return &JsonStore{filePath: filePath}
}

func (js *JsonStore) Set(key, value string) error {
	fmt.Println("Setting key value pair in kvstore")
	inputFile, err := os.Open(js.filePath)
	if err != nil {
		return fmt.Errorf("error opening file: %v", err)
	}

	defer func(inputFile *os.File) {
		err := inputFile.Close()
		if err != nil {
			fmt.Println("Error closing file:", err)
		}
	}(inputFile)

	byteValue, _ := io.ReadAll(inputFile)
	var data map[string]string
	err = json.Unmarshal(byteValue, &data)
	if err != nil {
		fmt.Println("Error unmarshalling JSON:", err)
		return err
	}

	data[key] = value

	output, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		fmt.Println("Error marshalling JSON:", err)
		return err
	}

	// Write the updated map to a new JSON file
	err = os.WriteFile(js.filePath, output, 0644)
	fmt.Println("successfully written to file")
	if err != nil {
		fmt.Println("Error writing file:", err)
		return err
	}
	return nil
}

func (js *JsonStore) Get(key string) (string, bool) {
	inputFile, err := os.Open(js.filePath)

	if err != nil {
		return "", false
	}

	defer func(inputFile *os.File) {
		err := inputFile.Close()
		if err != nil {
			fmt.Println("Error closing file:", err)
		}
	}(inputFile)

	byteValue, _ := io.ReadAll(inputFile)
	var data map[string]string
	err = json.Unmarshal(byteValue, &data)
	if err != nil {
		fmt.Println("Error unmarshalling JSON:", err)
		return "", false
	}

	return data[key], true
}
