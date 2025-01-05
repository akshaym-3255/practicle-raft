package raftnode

import (
	"akshay-raft/kvstore"
	"akshay-raft/logger"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"go.etcd.io/raft/v3/raftpb"
)

func loadSnapshot(dir string, store *kvstore.KeyValueStore) (*raftpb.Snapshot, error) {
	snapshotFiles, err := filepath.Glob(filepath.Join(dir, "*.snap"))
	if err != nil {
		return nil, err
	}

	if len(snapshotFiles) == 0 {
		logger.Log.Println("No snapshot found")
		return nil, nil
	}

	snapshotFile := snapshotFiles[len(snapshotFiles)-1]
	snapshotData, err := os.ReadFile(snapshotFile)
	if err != nil {
		return nil, err
	}

	var snapshot raftpb.Snapshot
	if err := snapshot.Unmarshal(snapshotData); err != nil {
		return nil, err
	}

	var state map[string]string
	if err := json.Unmarshal(snapshot.Data, &state); err != nil {
		return nil, fmt.Errorf("failed to restore application state: %v", err)
	}

	err = store.Restore(state)
	if err != nil {
		return nil, fmt.Errorf("failed to restore application state: %v", err)
	}

	logger.Log.Infof("Loaded snapshot: %s", snapshotFile)
	return &snapshot, nil
}

func saveSnapshot(snapshotDir string, snapshot raftpb.Snapshot) error {
	snapshotFile := filepath.Join(snapshotDir, fmt.Sprintf("snapshot-%d.snap", snapshot.Metadata.Index))

	// delete already existing snapshot file
	err := deletePreviousSnapshot(snapshotDir)

	data, err := snapshot.Marshal()
	if err != nil {
		return err
	}

	if err := os.WriteFile(snapshotFile, data, 0600); err != nil {
		return err
	}

	logger.Log.Infof("Saved snapshot at index: %d", snapshot.Metadata.Index)
	return nil
}

func deletePreviousSnapshot(dir string) error {
	snapshotFiles, err := filepath.Glob(filepath.Join(dir, "*.snap"))

	if err != nil {
		return err
	}
	for _, file := range snapshotFiles {
		err := os.Remove(file)
		if err != nil {
			return err
		}
	}
	return nil
}
