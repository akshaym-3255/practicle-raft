package kvstore

import (
	"encoding/json"
	"fmt"
	"github.com/akshaym-3255/practicle-raft/logger"
	"io"
	"os"
)

type JsonStore struct {
	filePath string
}

func handleFileClose(storeFile *os.File) {
	err := storeFile.Close()
	if err != nil {
		logger.Log.Errorln("Error closing file:", err)
	}
}

func NewJsonStore(filePath string) *JsonStore {
	return &JsonStore{filePath: filePath}
}

func (js *JsonStore) Set(key, value string) error {
	storeFile, err := os.OpenFile(js.filePath, os.O_RDWR|os.O_CREATE, 0644)
	defer handleFileClose(storeFile)
	if err != nil {
		return fmt.Errorf("error opening file: %v", err)
	}

	data, err := readData(storeFile, err)
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
	storeFile, err := os.Open(js.filePath)
	defer handleFileClose(storeFile)
	if err != nil {
		return "", false
	}

	data, err := readData(storeFile, err)
	if err != nil {
		return "", false
	}

	return data[key], true
}

func (js *JsonStore) Delete(key string) error {
	storeFile, err := os.Open(js.filePath)
	defer handleFileClose(storeFile)
	if err != nil {
		return err
	}

	data, err := readData(storeFile, err)
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
	storeFile, err := os.Open(js.filePath)
	defer handleFileClose(storeFile)
	if err != nil {
		return nil
	}

	data, err := readData(storeFile, err)
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

func readData(storeFile *os.File, err error) (map[string]string, error) {
	byteValue, _ := io.ReadAll(storeFile)
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
