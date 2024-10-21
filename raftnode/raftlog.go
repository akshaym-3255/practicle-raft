package raftnode

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"go.etcd.io/raft/v3"
	"go.etcd.io/raft/v3/raftpb"
)

// loadRaftLog loads the Raft logs from disk
func loadRaftLog(dir string, storage *raft.MemoryStorage) error {
	logFiles, err := filepath.Glob(filepath.Join(dir, "*.log"))
	if err != nil {
		return err
	}

	for _, logFile := range logFiles {
		file, err := os.Open(logFile)
		if err != nil {
			return fmt.Errorf("failed to open log file %s: %w", logFile, err)
		}
		defer file.Close()

		// Read each line (each line is a JSON-encoded log entry)
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			var logEntry map[string]interface{}
			if err := json.Unmarshal(scanner.Bytes(), &logEntry); err != nil {
				return fmt.Errorf("failed to unmarshal log entry from JSON in file %s: %w", logFile, err)
			}

			// Convert the JSON back into a raftpb.Entry
			entry := raftpb.Entry{
				Index: uint64(logEntry["index"].(float64)),                                 // Cast to uint64
				Term:  uint64(logEntry["term"].(float64)),                                  // Cast to uint64
				Type:  raftpb.EntryType(raftpb.EntryType_value[logEntry["type"].(string)]), // Convert string back to EntryType
				Data:  []byte(logEntry["data"].(string)),                                   // Convert data back to bytes
			}

			// Append the entry to the Raft storage
			if err := storage.Append([]raftpb.Entry{entry}); err != nil {
				return fmt.Errorf("failed to append log entry to Raft storage: %w", err)
			}

			log.Printf("Loaded log entry from file: %s (Index: %d, Term: %d)", logFile, entry.Index, entry.Term)

		}
	}
	return nil
}

func appendToLogFile(logDir string, entry raftpb.Entry) error {
	// Open the file in append mode, create if it doesn't exist
	file, err := os.OpenFile(logDir+"/node.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	// Serialize the log entry to JSON
	logEntry := map[string]interface{}{
		"index": entry.Index,
		"term":  entry.Term,
		"type":  entry.Type.String(), // Convert the EntryType to a string
		"data":  string(entry.Data),  // Assuming the data is a string, adjust for other formats
	}

	// Convert the log entry to JSON format
	jsonData, err := json.Marshal(logEntry)
	if err != nil {
		return err
	}

	// Append the JSON data to the file, followed by a newline for separation
	_, err = file.Write(append(jsonData, '\n'))
	if err != nil {
		return err
	}

	log.Printf("Appended log entry: %v", logEntry)
	return nil
}
