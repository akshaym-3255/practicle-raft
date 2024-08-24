package main

import (
	"context"
	"encoding/json"
	"flag"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"go.etcd.io/etcd/raft/v3"
	"go.etcd.io/etcd/raft/v3/raftpb"
)

type raftNode struct {
	id         uint64
	node       raft.Node
	storage    *raft.MemoryStorage
	transport  *httpTransport
	kvStore    *keyValueStore
	stopc      chan struct{}
	httpServer *http.Server
}

func newRaftNode(id uint64, kvStore *keyValueStore, clientListenURL, peerListenURL string, initialCluster string) *raftNode {
	storage := raft.NewMemoryStorage()
	c := &raft.Config{
		ID:              id,
		ElectionTick:    10,
		HeartbeatTick:   1,
		Storage:         storage,
		MaxInflightMsgs: 256,
		MaxSizePerMsg:   1024 * 1024,
	}

	peerURLs := strings.Split(initialCluster, ",")
	var raftPeers []raft.Peer
	for i, _ := range peerURLs {
		raftPeers = append(raftPeers, raft.Peer{ID: uint64(i + 1)})
	}

	n := raft.StartNode(c, raftPeers)
	transport := newHTTPTransport(id, peerURLs)

	rn := &raftNode{
		id:        id,
		node:      n,
		storage:   storage,
		transport: transport,
		kvStore:   kvStore,
		stopc:     make(chan struct{}),
	}
	rn.start(clientListenURL, peerListenURL)
	return rn
}

func (rn *raftNode) start(clientListenURL, peerListenURL string) {
	go rn.serveHTTP(clientListenURL, peerListenURL)
	go rn.run()
}

func (rn *raftNode) run() {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-rn.stopc:
			return
		case rd := <-rn.node.Ready():
			if err := rn.storage.Append(rd.Entries); err != nil {
				log.Fatal(err)
			}
			//log.Println(rd)
			rn.transport.send(rd.Messages)
			for _, entry := range rd.CommittedEntries {
				if entry.Type == raftpb.EntryNormal && len(entry.Data) > 0 {
					var cmd map[string]string
					if err := json.Unmarshal(entry.Data, &cmd); err == nil {
						for k, v := range cmd {
							rn.kvStore.set(k, v)
						}
					}
				}
			}
			rn.node.Advance()
		case msg := <-rn.transport.recvC:
			rn.node.Step(context.Background(), msg)
		case <-ticker.C:
			rn.node.Tick()
		}
	}
}

func stripHTTPPrefix(url string) string {
	return strings.TrimPrefix(url, "http://")
}

func (rn *raftNode) serveHTTP(clientListenURL, peerListenURL string) {
	r := mux.NewRouter()
	r.HandleFunc("/kv/{key}", rn.handleGet).Methods("GET")
	r.HandleFunc("/kv/{key}", rn.handleSet).Methods("PUT")
	r.HandleFunc("/raft", rn.transport.receive).Methods("POST")

	// Client server
	go func() {
		clientAddr := stripHTTPPrefix(clientListenURL)
		log.Printf("Starting client HTTP server on %s", clientAddr)
		if err := http.ListenAndServe(clientAddr, r); err != nil && err != http.ErrServerClosed {
			log.Fatalf("ListenAndServe(): %v", err)
		}
	}()

	// Peer server
	peerAddr := stripHTTPPrefix(peerListenURL)
	log.Printf("Starting peer HTTP server on %s", peerAddr)
	if err := http.ListenAndServe(peerAddr, r); err != nil && err != http.ErrServerClosed {
		log.Fatalf("ListenAndServe(): %v", err)
	}
}

func (rn *raftNode) handleGet(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	key := vars["key"]
	if value, ok := rn.kvStore.get(key); ok {
		json.NewEncoder(w).Encode(value)
	} else {
		http.NotFound(w, r)
	}
}

func (rn *raftNode) handleSet(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	key := vars["key"]
	var value map[string]string
	if err := json.NewDecoder(r.Body).Decode(&value); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	cmd := map[string]string{key: value[key]}
	data, err := json.Marshal(cmd)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	rn.node.Propose(context.TODO(), data)
	w.WriteHeader(http.StatusNoContent)
}

func main() {
	id := flag.Uint64("id", 1, "node ID")
	clientListenURLs := flag.String("listen-client-urls", "http://localhost:2379", "client listen URL")
	peerListenURLs := flag.String("listen-peer-urls", "http://localhost:2380", "peer listen URL")
	initialCluster := flag.String("initial-cluster", "1=http://localhost:2380,2=http://localhost:2382,3=http://localhost:2384", "initial cluster configuration")
	//peers := flag.String("peers", "http://localhost:2382,http://localhost:2384", "comma-separated list of peer URLs")

	flag.Parse()

	kvStore := newKeyValueStore()

	newRaftNode(*id, kvStore, *clientListenURLs, *peerListenURLs, *initialCluster)
	select {}
}
