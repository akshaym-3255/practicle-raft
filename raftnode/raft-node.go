package raftnode

import (
	"akshay-raft/kvstore"
	"akshay-raft/transport"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"go.etcd.io/raft/v3"
	"go.etcd.io/raft/v3/raftpb"
)

type RaftNode struct {
	id          uint64
	Node        raft.Node
	storage     *raft.MemoryStorage
	Transport   *transport.HttpTransport
	KvStore     kvstore.KeyValueStore
	stopc       chan struct{}
	httpServer  *http.Server
	snapshotDir string
	logDir      string
}

func NewRaftNode(id uint64, kvStore *kvstore.KeyValueStore, initialCluster string, snapshotDir, logDir string) *RaftNode {
	snapshot, err := loadSnapshot(snapshotDir)
	if err != nil {
		log.Fatalf("Error loading snapshot: %v", err)
	}

	// Create a storage for the Raft log and apply snapshot if found
	storage := raft.NewMemoryStorage()

	// Recovering the node by loading snapshot if exists
	if snapshot != nil {
		if err := storage.ApplySnapshot(*snapshot); err != nil {
			log.Fatalf("Error applying snapshot: %v", err)
		}
	}

	// Recovering the node's logs from the log directory
	if err := loadRaftLog(logDir, storage); err != nil {
		log.Fatalf("Error loading logs: %v", err)
	}

	c := &raft.Config{
		ID:                        id,
		ElectionTick:              10,
		HeartbeatTick:             1,
		Storage:                   storage,
		MaxInflightMsgs:           256,
		MaxSizePerMsg:             1024 * 1024,
		MaxUncommittedEntriesSize: 1 << 30,
	}

	peerURLs := strings.Split(initialCluster, ",")
	var raftPeers []raft.Peer
	for i, _ := range peerURLs {
		raftPeers = append(raftPeers, raft.Peer{ID: uint64(i + 1)})
	}

	n := raft.StartNode(c, raftPeers)
	tp := transport.NewHTTPTransport(id, peerURLs)

	rn := &RaftNode{
		id:          id,
		Node:        n,
		storage:     storage,
		Transport:   tp,
		KvStore:     *kvStore,
		stopc:       make(chan struct{}),
		snapshotDir: snapshotDir,
		logDir:      logDir,
	}
	return rn
}

func (rn *RaftNode) Run() {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-rn.stopc:
			return
		case rd := <-rn.Node.Ready():
			if err := rn.storage.Append(rd.Entries); err != nil {
				log.Fatal(err)
			}
			rn.Transport.Send(rd.Messages)
			//fmt.Println(len(rd.CommittedEntries))
			if len(rd.CommittedEntries) > 0 {
				rn.maybeTriggerSnapshot(rd.CommittedEntries[len(rd.CommittedEntries)-1].Index, 0)
			}

			rn.appendToLog(rd.Entries)
			//if err := rn.saveSnapshot(rd.Snapshot); err != nil {
			//	log.Fatalf("Error saving snapshot: %v", err)
			//}
			for _, entry := range rd.CommittedEntries {
				if entry.Type == raftpb.EntryNormal && len(entry.Data) > 0 {
					var cmd map[string]string
					if err := json.Unmarshal(entry.Data, &cmd); err == nil {
						for k, v := range cmd {
							rn.KvStore.Set(k, v)
						}
					}
				}
			}
			rn.Node.Advance()
		case msg := <-rn.Transport.RecvC:
			err := rn.Node.Step(context.Background(), msg)
			if err != nil {
				return
			}
		case <-ticker.C:
			rn.Node.Tick()
		}
	}
}

// Check if a snapshot needs to be triggered
func (rn *RaftNode) maybeTriggerSnapshot(appliedIndex uint64, lastSnapshotIndex uint64) {
	// Define when you want to take a snapshot, e.g., every 1000 log entries
	snapshotThreshold := uint64(1000)
	fmt.Println("appliedIndex", appliedIndex)
	if appliedIndex-lastSnapshotIndex >= snapshotThreshold {
		// Trigger a snapshot
		log.Printf("Triggering snapshot at applied index: %d", appliedIndex)
		rn.createSnapshot(appliedIndex)
		lastSnapshotIndex = appliedIndex
	}
}

// Create a snapshot of the current state and persist it
func (rn *RaftNode) createSnapshot(appliedIndex uint64) {
	// The snapshot includes the index and the state of your application
	snapshot := raftpb.Snapshot{
		Metadata: raftpb.SnapshotMetadata{
			Index: appliedIndex,
			Term:  4, // The term at the time of the snapshot
		},
		//Data: serializeStateMachineState(), // Serialize your application's state here
	}

	// Save the snapshot to disk
	if err := rn.saveSnapshot(snapshot); err != nil {
		log.Fatalf("Failed to save snapshot: %v", err)
	}

	// Compact the log (i.e., discard entries up to the snapshot index)
	//node.Compact(appliedIndex)
}

func loadSnapshot(dir string) (*raftpb.Snapshot, error) {
	snapshotFiles, err := filepath.Glob(filepath.Join(dir, "*.snap"))
	if err != nil {
		return nil, err
	}

	if len(snapshotFiles) == 0 {
		log.Println("No snapshot found")
		return nil, nil
	}

	// Load the latest snapshot
	snapshotFile := snapshotFiles[len(snapshotFiles)-1]
	snapshotData, err := os.ReadFile(snapshotFile)
	if err != nil {
		return nil, err
	}

	var snapshot raftpb.Snapshot
	if err := snapshot.Unmarshal(snapshotData); err != nil {
		return nil, err
	}

	log.Printf("Loaded snapshot: %s", snapshotFile)
	return &snapshot, nil
}

// loadRaftLog loads the Raft logs from disk
func loadRaftLog(dir string, storage *raft.MemoryStorage) error {
	logFiles, err := filepath.Glob(filepath.Join(dir, "*.log"))
	if err != nil {
		return err
	}

	fmt.Println(logFiles)
	for _, logFile := range logFiles {
		logData, err := os.ReadFile(logFile)
		fmt.Println(string(logData))

		if err != nil {
			return err
		}

		var entry raftpb.Entry
		if err := entry.Unmarshal(logData); err != nil {
			return err
		}

		if err := storage.Append([]raftpb.Entry{entry}); err != nil {
			return err
		}
		log.Printf("Loaded log entry: %s", logFile)
	}

	return nil
}

// Save the snapshot to disk
func (rn *RaftNode) saveSnapshot(snapshot raftpb.Snapshot) error {
	snapshotFile := filepath.Join(rn.snapshotDir, fmt.Sprintf("snapshot-%d.snap", snapshot.Metadata.Index))
	data, err := snapshot.Marshal()
	if err != nil {
		return err
	}

	if err := os.WriteFile(snapshotFile, data, 0600); err != nil {
		return err
	}

	log.Printf("Saved snapshot at index: %d", snapshot.Metadata.Index)
	return nil
}

func (rn *RaftNode) appendToLog(entries []raftpb.Entry) {
	for _, entry := range entries {
		if entry.Type == raftpb.EntryNormal {
			log.Printf("Received entry: %s", entry.Data)
			err := rn.appendToLogFile(entry)
			if err != nil {
				return
			}
		}
	}

}

// applyLogEntry applies a committed log entry to the state machine
func (rn *RaftNode) appendToLogFile(entry raftpb.Entry) error {
	// Open the file in append mode, create if it doesn't exist
	file, err := os.OpenFile(rn.logDir+"/node.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
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

//func (rn *RaftNode) start(clientListenURL, peerListenURL string) {
//
//	go peerServer.ServeHTTP()
//	go rn.serveHTTP(clientListenURL, peerListenURL)
//	go rn.run()
//}
