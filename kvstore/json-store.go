package kvstore

import (
	"akshay-raft/logger"
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
		return fmt.Errorf("error opening file: %v", err)
	}

	defer func(inputFile *os.File) {
		err := inputFile.Close()
		if err != nil {
			logger.Log.Errorln("Error closing file:", err)
		}
	}(inputFile)

	data, err := readData(inputFile, err)
	if err != nil {
		return err
	}

	data[key] = value

	err = writeData(data, js.filePath)
	if err != nil {
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
			logger.Log.Errorln("Error closing file:", err)
		}
	}(inputFile)

	data, err := readData(inputFile, err)
	if err != nil {
		return "", false
	}

	return data[key], true
}

func (js *JsonStore) Delete(key string) error {
	inputFile, err := os.Open(js.filePath)
	if err != nil {
		return err
	}

	defer func(inputFile *os.File) {
		err := inputFile.Close()
		if err != nil {
			logger.Log.Errorln("Error closing file:", err)
		}
	}(inputFile)

	data, err := readData(inputFile, err)
	if err != nil {
		return err
	}

	delete(data, key)

	err = writeData(data, js.filePath)
	if err != nil {
		return err
	}
	return nil
}

func (js *JsonStore) Dump() map[string]string {
	inputFile, err := os.Open(js.filePath)

	if err != nil {
		return nil
	}

	defer func(inputFile *os.File) {
		err := inputFile.Close()
		if err != nil {
			logger.Log.Errorln("Error closing file:", err)
		}
	}(inputFile)

	data, err := readData(inputFile, err)
	if err != nil {
		return nil
	}

	return data
}

func (js *JsonStore) Restore(data map[string]string) error {
	err := writeData(data, js.filePath)
	if err != nil {
		return err
	}
	return nil
}

func writeData(data map[string]string, filePath string) error {
	output, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		logger.Log.Errorln("Error marshalling JSON:", err)
		return err
	}
	err = os.WriteFile(filePath, output, 0644)
	if err != nil {
		logger.Log.Errorln("Error writing file:", err)
		return err
	}
	return nil
}

func readData(inputFile *os.File, err error) (map[string]string, error) {
	byteValue, _ := io.ReadAll(inputFile)
	if len(byteValue) == 0 {
		byteValue = []byte("{}")

	}
	var data map[string]string
	err = json.Unmarshal(byteValue, &data)
	if err != nil {
		return nil, err
	}
	return data, nil
}
