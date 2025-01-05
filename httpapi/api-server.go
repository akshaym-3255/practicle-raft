package httpapi

import (
	"akshay-raft/raftnode"
	"context"
	"encoding/json"
	"errors"
	"github.com/gorilla/mux"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type ApiServer struct {
	RaftNode *raftnode.RaftNode
}

func stripHTTPPrefix(url string) string {
	return strings.TrimPrefix(url, "http://")
}

func (as *ApiServer) ServeHTTP(clientListenURL string) {
	r := mux.NewRouter()
	r.HandleFunc("/kv/{key}", as.handleGet).Methods("GET")
	r.HandleFunc("/kv/{key}", as.handleSet).Methods("PUT")
	r.HandleFunc("/add-node", as.addNodeHandler).Methods("POST")
	r.HandleFunc("/add-node", as.removeNodeHandler).Methods("DELETE")

	clientAddr := stripHTTPPrefix(clientListenURL)
	log.Printf("Starting client HTTP server on %s", clientAddr)
	if err := http.ListenAndServe(clientAddr, r); err != nil && err != http.ErrServerClosed {
		log.Fatalf("ListenAndServe(): %v", err)
	}

}

func (as *ApiServer) handleGet(w http.ResponseWriter, r *http.Request) {

	vars := mux.Vars(r)
	key := vars["key"]
	if key == "" {
		http.Error(w, "Key is required", http.StatusBadRequest)
		return
	}

	if as.RaftNode.Node.Status().Lead == as.RaftNode.Id {
		value, ok := as.RaftNode.KvStore.Get(key)

		if !ok {
			http.Error(w, "Key not found", http.StatusNotFound)
			return
		}

		json.NewEncoder(w).Encode(map[string]string{key: value})
		return
	}

	reqCtx := []byte(key)
	ctx := context.TODO()

	log.Printf("readIndex started")
	err := as.RaftNode.Node.ReadIndex(ctx, reqCtx)
	if err != nil {
		http.Error(w, "Failed to initiate ReadIndex", http.StatusInternalServerError)
		return
	}

	rs := <-as.RaftNode.ReadState

	if string(rs.RequestCtx) != key {
		http.Error(w, "ReadIndex mismatch", http.StatusInternalServerError)
		return
	}

	// wait for 10 seconds
	wait := 0
	for as.RaftNode.CommitIndex < rs.Index {
		if wait > 10 {
			break
		}
		time.Sleep(1 * time.Second)
		wait += 1
	}

	value, ok := as.RaftNode.KvStore.Get(key)

	if !ok {
		http.Error(w, "Key not found", http.StatusNotFound)
		return
	}

	json.NewEncoder(w).Encode(map[string]string{key: value})

	//// Check if this node is the leader
	//if as.RaftNode.Node.Status().Lead != as.RaftNode.Id {
	//	// Not the leader, redirect the request to the leader node
	//	leaderID := as.RaftNode.Node.Status().Lead
	//	leaderAddress := as.RaftNode.Transport.GetPeerURL(leaderID)
	//	if leaderAddress != "" {
	//		http.Redirect(w, r, "http://"+leaderAddress+r.RequestURI, http.StatusTemporaryRedirect)
	//		return
	//	} else {
	//		http.Error(w, "Leader unknown", http.StatusInternalServerError)
	//		return
	//	}
	//}
	//
	//// This node is the leader, process the read request

	//if value, ok := as.RaftNode.KvStore.Get(key); ok {
	//	json.NewEncoder(w).Encode(value)
	//} else {
	//	http.NotFound(w, r)
	//}
}

func (as *ApiServer) handleSet(w http.ResponseWriter, r *http.Request) {
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

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = as.RaftNode.Node.Propose(ctx, data)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			http.Error(w, "Propose operation timed out", http.StatusRequestTimeout)
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	// If Propose succeeds, respond with no content
	w.WriteHeader(http.StatusNoContent)
}

func (as *ApiServer) addNodeHandler(w http.ResponseWriter, r *http.Request) {
	nodeIDStr := r.URL.Query().Get("node_id")
	nodeID, err := strconv.ParseUint(nodeIDStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid node ID", http.StatusBadRequest)
		return
	}
	nodeURL := r.URL.Query().Get("node_url")
	if nodeURL == "" {
		http.Error(w, "Missing node URL", http.StatusBadRequest)
		return
	}

	err = as.RaftNode.AddNode(nodeID, nodeURL)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	log.Printf("Sucessfully added node with NodeId %d", nodeID)
}

func (as *ApiServer) removeNodeHandler(w http.ResponseWriter, r *http.Request) {
	nodeIDStr := r.URL.Query().Get("node_id")
	nodeID, err := strconv.ParseUint(nodeIDStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid node ID", http.StatusBadRequest)
		return
	}

	err = as.RaftNode.RemoveNode(nodeID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	log.Printf("Sucessfully removed node with NodeId %d", nodeID)

}
