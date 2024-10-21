package raftnode

import (
	"akshay-raft/kvstore"
	"akshay-raft/transport"
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"go.etcd.io/raft/v3"
	"go.etcd.io/raft/v3/raftpb"
)

type RaftNode struct {
	id                uint64
	Node              raft.Node
	storage           *raft.MemoryStorage
	Transport         *transport.HttpTransport
	KvStore           kvstore.KeyValueStore
	ConfState         raftpb.ConfState
	stopc             chan struct{}
	httpServer        *http.Server
	snapshotDir       string
	logDir            string
	lastSnapshotIndex uint64
}

func NewRaftNode(id uint64, kvStore *kvstore.KeyValueStore, initialCluster string, snapshotDir, logDir string, join bool) *RaftNode {
	snapshot, err := loadSnapshot(snapshotDir, kvStore)
	if err != nil {
		log.Fatalf("Error loading snapshot: %v", err)
	}

	// Create a storage for the Raft log and apply snapshot if found
	storage := raft.NewMemoryStorage()

	var lastSnapshotIndex uint64
	var confState raftpb.ConfState
	// Recovering the node by loading snapshot if exists
	if snapshot != nil {
		if err := storage.ApplySnapshot(*snapshot); err != nil {
			log.Fatalf("Error applying snapshot: %v", err)
		}
		confState = snapshot.Metadata.ConfState
		lastSnapshotIndex = snapshot.Metadata.Index
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

	var n raft.Node
	if join {
		n = raft.RestartNode(c)
	} else {
		n = raft.StartNode(c, raftPeers)
	}

	tp := transport.NewHTTPTransport(id, peerURLs)

	rn := &RaftNode{
		id:                id,
		Node:              n,
		storage:           storage,
		Transport:         tp,
		KvStore:           *kvStore,
		stopc:             make(chan struct{}),
		snapshotDir:       snapshotDir,
		logDir:            logDir,
		lastSnapshotIndex: lastSnapshotIndex,
	}
	rn.ConfState = confState
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
			if len(rd.CommittedEntries) > 0 {
				rn.maybeTriggerSnapshot(rd.CommittedEntries[len(rd.CommittedEntries)-1].Index)
			}

			rn.appendToLog(rd.Entries)

			for _, entry := range rd.CommittedEntries {
				if entry.Type == raftpb.EntryNormal && len(entry.Data) > 0 {
					var cmd map[string]string
					if err := json.Unmarshal(entry.Data, &cmd); err == nil {
						for k, v := range cmd {
							rn.KvStore.Set(k, v)
						}
					}
				}

				if entry.Type == raftpb.EntryConfChange {
					var cc raftpb.ConfChange
					if err := cc.Unmarshal(entry.Data); err != nil {
						log.Fatalf("failed to unmarshal conf change: %v", err)
					}
					rn.ConfState = *rn.Node.ApplyConfChange(cc)
					rn.Transport.AddPeer(cc.NodeID, string(cc.Context))
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

func (rn *RaftNode) maybeTriggerSnapshot(appliedIndex uint64) {
	snapshotThreshold := uint64(10)
	if appliedIndex-rn.lastSnapshotIndex >= snapshotThreshold {
		log.Printf("Triggering snapshot at applied index: %d", appliedIndex)
		rn.createSnapshot(appliedIndex)
		rn.lastSnapshotIndex = appliedIndex
	}
}

func (rn *RaftNode) createSnapshot(appliedIndex uint64) {

	kvStateSnapData, err := json.Marshal(rn.KvStore.Dump())

	if err != nil {
		log.Fatalf("Failed to serialize state for snapshot: %v", err)
	}

	snapshot := raftpb.Snapshot{
		Metadata: raftpb.SnapshotMetadata{
			Index:     appliedIndex,
			Term:      rn.Node.Status().Term,
			ConfState: rn.ConfState,
		},
		Data: kvStateSnapData,
	}

	if err := saveSnapshot(rn.snapshotDir, snapshot); err != nil {
		log.Fatalf("Failed to save snapshot: %v", err)
	}
	if err := rn.storage.Compact(appliedIndex); err != nil {
		log.Fatalf("Failed to compact Raft logs: %v", err)
	}
	err = compactLogFile(rn.logDir, appliedIndex)
	if err != nil {
		log.Fatalf("Failed to compact node.log file: %v", err)
	}

	log.Printf("Snapshot created at index: %d, term: %d", snapshot.Metadata.Index, snapshot.Metadata.Term)
}

func (rn *RaftNode) appendToLog(entries []raftpb.Entry) {
	for _, entry := range entries {
		err := appendToLogFile(rn.logDir, entry)
		if err != nil {
			return
		}

	}
}

func (rn *RaftNode) AddNode(newNodeID uint64, newNodeURL string) error {
	log.Printf("Adding new node with ID %d, URL: %s", newNodeID, newNodeURL)

	cc := raftpb.ConfChange{
		Type:    raftpb.ConfChangeAddNode,
		NodeID:  newNodeID,
		Context: []byte(newNodeURL),
	}

	err := rn.Node.ProposeConfChange(context.TODO(), cc)
	if err != nil {
		return fmt.Errorf("failed to propose conf change: %v", err)
	}

	rn.Transport.AddPeer(newNodeID, newNodeURL)

	log.Printf("Proposed configuration change to add node %d", newNodeID)
	return nil
}

// compactLogFile removes log entries before the given appliedIndex from the log file
func compactLogFile(logDir string, appliedIndex uint64) error {
	// Open the log file
	logFile, err := os.OpenFile(logDir+"/node.log", os.O_RDWR, 0644)
	if err != nil {
		return fmt.Errorf("failed to open log file for compaction: %v", err)
	}
	defer logFile.Close()

	// Scanner for reading the log file line-by-line
	scanner := bufio.NewScanner(logFile)
	var newLogEntries []raftpb.Entry

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
		// Only keep entries after or at the appliedIndex (since older entries are in the snapshot)
		if entry.Index >= appliedIndex {
			newLogEntries = append(newLogEntries, entry)
		}
	}

	// Truncate and rewrite the log file with only the new entries
	if err := logFile.Truncate(0); err != nil {
		return fmt.Errorf("failed to truncate log file: %v", err)
	}
	if _, err := logFile.Seek(0, 0); err != nil {
		return fmt.Errorf("failed to seek log file: %v", err)
	}

	writer := bufio.NewWriter(logFile)
	for _, entry := range newLogEntries {
		err = appendToLogFile(logDir, entry)
		if err != nil {
			return fmt.Errorf("failed to encode log entry: %v", err)
		}
	}

	if err := writer.Flush(); err != nil {
		return fmt.Errorf("failed to flush buffer: %v", err)
	}

	log.Printf("Log file compacted, removed entries before index: %d", appliedIndex)
	return nil

}
