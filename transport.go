package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"

	"go.etcd.io/etcd/raft/v3/raftpb"
)

type httpTransport struct {
	id      uint64
	peers   []string
	client  *http.Client
	peerMap map[string]uint64
	recvC   chan raftpb.Message
	mu      sync.Mutex
}

func newHTTPTransport(id uint64, peers []string) *httpTransport {
	peerMap := make(map[string]uint64)
	for _, peer := range peers {
		idHost := strings.Split(peer, "=")
		id, _ := strconv.ParseInt(idHost[0], 10, 64)
		peerMap[idHost[1]] = uint64(id)
	}
	return &httpTransport{
		id:      id,
		peers:   peers,
		client:  &http.Client{},
		peerMap: peerMap,
		recvC:   make(chan raftpb.Message, 1024),
	}
}

func (t *httpTransport) send(messages []raftpb.Message) {
	for _, msg := range messages {
		if msg.To == t.id {
			t.recvC <- msg
		} else {
			t.sendMessage(msg)
		}
	}
}

func (t *httpTransport) sendMessage(msg raftpb.Message) {
	t.mu.Lock()
	defer t.mu.Unlock()
	data, err := msg.Marshal()
	if err != nil {
		log.Printf("failed to marshal message: %v", err)
		return
	}
	peerURL := t.getPeerURL(msg.To)
	if peerURL == "" {
		log.Printf("failed to find peer URL for node %d", msg.To)
		return
	}
	url := fmt.Sprintf("%s/raft", peerURL)
	resp, err := t.client.Post(url, "application/octet-stream", bytes.NewReader(data))
	if err != nil {
		log.Printf("failed to send message to %s: %v", url, err)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		log.Printf("failed to send message to %s, status: %v", url, resp.Status)
	}
}

func (t *httpTransport) getPeerURL(id uint64) string {
	for peer, peerID := range t.peerMap {
		if peerID == id {
			return peer
		}
	}
	return ""
}

func (t *httpTransport) receive(w http.ResponseWriter, r *http.Request) {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var msg raftpb.Message
	if err := msg.Unmarshal(body); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	t.recvC <- msg
	w.WriteHeader(http.StatusOK)
}
