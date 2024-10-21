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
	inputFile, err := os.OpenFile(js.filePath, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		fmt.Println(err)
		return fmt.Errorf("error opening file: %v", err)
	}

	defer func(inputFile *os.File) {
		err := inputFile.Close()
		if err != nil {
			fmt.Println("Error closing file:", err)
		}
	}(inputFile)

	byteValue, _ := io.ReadAll(inputFile)
	if len(byteValue) == 0 {
		byteValue = []byte("{}")

	}
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

func (js *JsonStore) Dump() map[string]string {
	inputFile, err := os.Open(js.filePath)

	if err != nil {
		return nil
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
		return nil
	}

	return data
}

func (js *JsonStore) Restore(data map[string]string) error {
	output, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		fmt.Println("Error marshalling JSON:", err)
		return err
	}

	err = os.WriteFile(js.filePath, output, 0644)
	if err != nil {
		fmt.Println("Error writing file:", err)
		return err
	}
	return nil
}
