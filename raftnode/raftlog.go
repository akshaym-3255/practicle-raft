package raftnode

import (
	"bufio"
	"encoding/json"
	"fmt"
	"github.com/akshaym-3255/practicle-raft/logger"
	"os"
	"path/filepath"

	"go.etcd.io/raft/v3"
	"go.etcd.io/raft/v3/raftpb"
)

func loadRaftLog(dir string, storage *raft.MemoryStorage) error {
	logFiles, err := filepath.Glob(filepath.Join(dir, "*.log"))
	if err != nil {
		return err
	}

	for _, logFile := range logFiles {
		file, err := os.Open(logFile)
		if err != nil {
			return fmt.Errorf("failed to open logger file %s: %w", logFile, err)
		}
		defer file.Close()

		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			var logEntry map[string]interface{}
			if err := json.Unmarshal(scanner.Bytes(), &logEntry); err != nil {
				return fmt.Errorf("failed to unmarshal logger entry from JSON in file %s: %w", logFile, err)
			}

			entry := raftpb.Entry{
				Index: uint64(logEntry["index"].(float64)),
				Term:  uint64(logEntry["term"].(float64)),
				Type:  raftpb.EntryType(raftpb.EntryType_value[logEntry["type"].(string)]),
				Data:  []byte(logEntry["data"].(string)),
			}

			if err := storage.Append([]raftpb.Entry{entry}); err != nil {
				return fmt.Errorf("failed to append logger entry to Raft storage: %w", err)
			}

			logger.Log.Infof("Loaded logger entry from file: %s (Index: %d, Term: %d)", logFile, entry.Index, entry.Term)
		}
	}
	return nil
}

func appendToLogFile(logDir string, entry raftpb.Entry) error {
	file, err := os.OpenFile(logDir+"/node.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	logEntry := map[string]interface{}{
		"index": entry.Index,
		"term":  entry.Term,
		"type":  entry.Type.String(),
		"data":  string(entry.Data),
	}

	jsonData, err := json.Marshal(logEntry)
	if err != nil {
		return err
	}

	_, err = file.Write(append(jsonData, '\n'))
	if err != nil {
		return err
	}

	logger.Log.Infof("Appended logger entry: %v", logEntry)
	return nil
}

func compactLogFile(logDir string, appliedIndex uint64) error {
	logFile, err := os.OpenFile(logDir+"/node.log", os.O_RDWR, 0644)
	if err != nil {
		return fmt.Errorf("failed to open logger file for compaction: %v", err)
	}
	defer logFile.Close()

	scanner := bufio.NewScanner(logFile)
	var newLogEntries []raftpb.Entry

	for scanner.Scan() {
		var logEntry map[string]interface{}
		if err := json.Unmarshal(scanner.Bytes(), &logEntry); err != nil {
			return fmt.Errorf("failed to unmarshal logger entry from JSON in file %s: %w", logFile.Name(), err)
		}

		entry := raftpb.Entry{
			Index: uint64(logEntry["index"].(float64)),
			Term:  uint64(logEntry["term"].(float64)),
			Type:  raftpb.EntryType(raftpb.EntryType_value[logEntry["type"].(string)]),
			Data:  []byte(logEntry["data"].(string)),
		}
		if entry.Index >= appliedIndex {
			newLogEntries = append(newLogEntries, entry)
		}
	}

	if err := logFile.Truncate(0); err != nil {
		return fmt.Errorf("failed to truncate logger file: %v", err)
	}
	if _, err := logFile.Seek(0, 0); err != nil {
		return fmt.Errorf("failed to seek logger file: %v", err)
	}

	writer := bufio.NewWriter(logFile)
	for _, entry := range newLogEntries {
		err = appendToLogFile(logDir, entry)
		if err != nil {
			return fmt.Errorf("failed to encode logger entry: %v", err)
		}
	}

	if err := writer.Flush(); err != nil {
		return fmt.Errorf("failed to flush buffer: %v", err)
	}

	logger.Log.Infof("Log file compacted, removed entries before index: %d", appliedIndex)
	return nil
}
