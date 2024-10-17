package httpapi

import (
	"akshay-raft/raftnode"
	"github.com/gorilla/mux"
	"log"
	"net/http"
)

type PeerServer struct {
	RaftNode *raftnode.RaftNode
}

func (ps *PeerServer) ServeHTTP(peerListenURL string) {
	r := mux.NewRouter()
	r.HandleFunc("/raft", ps.RaftNode.Transport.Receive).Methods("POST")

	peerAddr := stripHTTPPrefix(peerListenURL)
	log.Printf("Starting peer HTTP server on %s", peerAddr)
	if err := http.ListenAndServe(peerAddr, r); err != nil && err != http.ErrServerClosed {
		log.Fatalf("ListenAndServe(): %v", err)
	}
}
