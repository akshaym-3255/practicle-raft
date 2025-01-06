package raftnode

import (
	"akshay-raft/kvstore"
	"akshay-raft/logger"
	"akshay-raft/transport"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"go.etcd.io/raft/v3"
	"go.etcd.io/raft/v3/raftpb"
)

// RaftNode is a wrapper around the core Raft.Node, encapsulating additional functionalities and state.
type RaftNode struct {
	// Id is the unique identifier for this node in the Raft cluster.
	Id uint64

	// Node represents the core Raft instance for this node,
	// managing consensus and state replication.
	Node raft.Node

	// storage is an in-memory storage used by the Raft library
	// to store logs, snapshots, and metadata.
	storage *raft.MemoryStorage

	// Transport handles network communication between nodes in the Raft cluster,
	// using HTTP as the transport protocol.
	Transport *transport.HttpTransport

	// KvStore represents the key-value store that
	// holds the application state replicated across the cluster.
	KvStore kvstore.KeyValueStore

	// ConfState holds the current configuration state of the cluster,
	// including members and their roles.
	ConfState raftpb.ConfState

	// ReadState is a channel for receiving linearizable read states
	// (`raft.ReadState`) requested by clients.
	ReadState chan raft.ReadState

	// stopc is a channel used to signal the Raft node to shut down gracefully.
	stopc chan struct{}

	// httpServer is the HTTP server instance
	// used for exposing APIs and interacting with clients.
	httpServer *http.Server

	// snapshotDir is the directory where snapshots of the Raft state machine
	// are stored for recovery purposes.
	snapshotDir string

	// logDir is the directory where Raft logger files
	// are stored for durability and replaying state.
	logDir string

	// lastSnapshotIndex is the index of the
	// last snapshot applied, used for logger compaction and recovery.
	lastSnapshotIndex uint64

	// CommitIndex is the highest logger entry index
	// that has been committed to the state machine.
	CommitIndex uint64
}

type LogDataEntry struct {
	Operation string `json:"operation"`
	Key       string `json:"key"`
	Value     string `json:"value"`
}

func NewRaftNode(id uint64, kvStore *kvstore.KeyValueStore, initialCluster string, snapshotDir, logDir string, join bool) *RaftNode {

	loggerRaft := logrus.New()
	loggerRaft.SetLevel(logrus.DebugLevel)
	raft.SetLogger(loggerRaft)
	snapshot, err := loadSnapshot(snapshotDir, kvStore)
	if err != nil {
		logger.Log.Fatalf("Error loading snapshot: %v", err)
	}

	// Create a storage for the Raft logger and apply snapshot if found
	storage := raft.NewMemoryStorage()

	var lastSnapshotIndex uint64
	var confState raftpb.ConfState
	// Recovering the node by loading snapshot if exists
	if snapshot != nil {
		if err := storage.ApplySnapshot(*snapshot); err != nil {
			logger.Log.Fatalf("Error applying snapshot: %v", err)
		}
		confState = snapshot.Metadata.ConfState
		lastSnapshotIndex = snapshot.Metadata.Index
	}

	// Recovering the node's logs from the logger directory
	logger.Log.Infof("recovering from logs")
	if err := loadRaftLog(logDir, storage); err != nil {
		logger.Log.Fatalf("Error loading logs: %v", err)
	}

	c := &raft.Config{
		ID:                        id,
		ElectionTick:              100,
		HeartbeatTick:             10,
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
		logger.Log.Infof("restarting Node")
		n = raft.RestartNode(c)
	} else {
		n = raft.StartNode(c, raftPeers)
	}

	tp := transport.NewHTTPTransport(id, peerURLs)

	rn := &RaftNode{
		Id:                id,
		Node:              n,
		storage:           storage,
		Transport:         tp,
		KvStore:           *kvStore,
		stopc:             make(chan struct{}),
		ReadState:         make(chan raft.ReadState),
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

			//
			for _, rs := range rd.ReadStates {
				rn.ReadState <- rs
			}

			if err := rn.storage.Append(rd.Entries); err != nil {
				logger.Log.Fatal(err)
			}

			rn.Transport.Send(rd.Messages)

			if len(rd.CommittedEntries) > 0 {
				//to demo readState
				if rn.Id == 5 {
					time.Sleep(30 * time.Second)
				}
				rn.CommitIndex = rd.CommittedEntries[len(rd.CommittedEntries)-1].Index
				rn.maybeTriggerSnapshot(rd.CommittedEntries[len(rd.CommittedEntries)-1].Index)
				rn.appendToLog(rd.CommittedEntries)
			}

			for _, entry := range rd.CommittedEntries {

				if entry.Type == raftpb.EntryNormal && len(entry.Data) > 0 {
					var logDataEntry LogDataEntry
					if err := json.Unmarshal(entry.Data, &logDataEntry); err == nil {
						if logDataEntry.Operation == "Add" {
							rn.KvStore.Set(logDataEntry.Key, logDataEntry.Value)
						} else if logDataEntry.Operation == "Delete" {
							rn.KvStore.Delete(logDataEntry.Key)
						}
					}
				}

				if entry.Type == raftpb.EntryConfChange {
					var cc raftpb.ConfChange
					if err := cc.Unmarshal(entry.Data); err != nil {
						logger.Log.Fatalf("failed to unmarshal conf change: %v", err)
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
	snapshotThreshold := uint64(1000)
	if appliedIndex-rn.lastSnapshotIndex >= snapshotThreshold {
		logger.Log.Infof("Triggering snapshot at applied index: %d", appliedIndex)
		rn.createSnapshot(appliedIndex)
		rn.lastSnapshotIndex = appliedIndex
	}
}

func (rn *RaftNode) createSnapshot(appliedIndex uint64) {

	kvStateSnapData, err := json.Marshal(rn.KvStore.Dump())

	if err != nil {
		logger.Log.Fatalf("Failed to serialize state for snapshot: %v", err)
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
		logger.Log.Fatalf("Failed to save snapshot: %v", err)
	}
	if err := rn.storage.Compact(appliedIndex); err != nil {
		logger.Log.Fatalf("Failed to compact Raft logs: %v", err)
	}
	err = compactLogFile(rn.logDir, appliedIndex)
	if err != nil {
		logger.Log.Fatalf("Failed to compact node.logger file: %v", err)
	}

	logger.Log.Infof("Snapshot created at index: %d, term: %d", snapshot.Metadata.Index, snapshot.Metadata.Term)
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
	logger.Log.Infof("Adding new node with ID %d, URL: %s", newNodeID, newNodeURL)

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

	logger.Log.Infof("Proposed configuration change to add node %d", newNodeID)
	return nil
}

func (rn *RaftNode) RemoveNode(nodeId uint64) error {
	logger.Log.Infof("Removing new node with ID %d", nodeId)

	cc := raftpb.ConfChange{
		Type:    raftpb.ConfChangeRemoveNode,
		NodeID:  nodeId,
		Context: []byte("deleteNode"),
	}

	err := rn.Node.ProposeConfChange(context.TODO(), cc)
	if err != nil {
		return fmt.Errorf("failed to propose conf change: %v", err)
	}

	rn.Transport.RemovePeer(nodeId)

	logger.Log.Infof("Proposed configuration change to remove node %d", nodeId)
	return nil
}
