package httpapi

import (
	"akshay-raft/raftnode"
	"context"
	"encoding/json"
	"fmt"
	"github.com/gorilla/mux"
	"log"
	"net/http"
	"strings"
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
	fmt.Println("Proposing data")
	err = as.RaftNode.Node.Propose(context.TODO(), data)
	if err != nil {
		fmt.Println(err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
