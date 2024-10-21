package httpapi

import (
	"akshay-raft/raftnode"
	"context"
	"encoding/json"
	"errors"
	"fmt"
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

	clientAddr := stripHTTPPrefix(clientListenURL)
	log.Printf("Starting client HTTP server on %s", clientAddr)
	if err := http.ListenAndServe(clientAddr, r); err != nil && err != http.ErrServerClosed {
		log.Fatalf("ListenAndServe(): %v", err)
	}

}

func (as *ApiServer) handleGet(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	key := vars["key"]
	if value, ok := as.RaftNode.KvStore.Get(key); ok {
		json.NewEncoder(w).Encode(value)
	} else {
		http.NotFound(w, r)
	}
}

func (as *ApiServer) handleSet(w http.ResponseWriter, r *http.Request) {
	fmt.Println("handleSet")
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

	// Propose the data using the context
	fmt.Println("Proposing data")
	err = as.RaftNode.Node.Propose(ctx, data)
	if err != nil {
		// Handle timeout or other errors from Propose
		if errors.Is(err, context.DeadlineExceeded) {
			fmt.Println("Propose operation timed out")
			http.Error(w, "Propose operation timed out", http.StatusRequestTimeout)
		} else {
			fmt.Println("Propose operation failed:", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	// If Propose succeeds, respond with no content
	w.WriteHeader(http.StatusNoContent)
}

func (api *ApiServer) addNodeHandler(w http.ResponseWriter, r *http.Request) {
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

	err = api.RaftNode.AddNode(nodeID, nodeURL)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	fmt.Fprintf(w, "Node %d added successfully", nodeID)
}
