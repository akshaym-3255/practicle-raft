package transport

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"

	"go.etcd.io/raft/v3/raftpb"
)

type HttpTransport struct {
	id      uint64
	peers   []string
	client  *http.Client
	peerMap map[string]uint64
	RecvC   chan raftpb.Message
	mu      sync.Mutex
}

func NewHTTPTransport(id uint64, peers []string) *HttpTransport {
	peerMap := make(map[string]uint64)
	for _, peer := range peers {
		idHost := strings.Split(peer, "=")
		id, _ := strconv.ParseInt(idHost[0], 10, 64)
		peerMap[idHost[1]] = uint64(id)
	}
	return &HttpTransport{
		id:      id,
		peers:   peers,
		client:  &http.Client{},
		peerMap: peerMap,
		RecvC:   make(chan raftpb.Message, 1024),
	}
}

func (t *HttpTransport) AddPeer(newNodeID uint64, newPeerURL string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.peerMap[newPeerURL] = newNodeID
	t.peers = append(t.peers, newPeerURL)
	log.Printf("Added new peer: Node ID %d, URL: %s", newNodeID, newPeerURL)
}

func (t *HttpTransport) Send(messages []raftpb.Message) {
	for _, msg := range messages {
		if msg.To == t.id {
			t.RecvC <- msg
		} else {
			t.SendMessage(msg)
		}
	}
}

func (t *HttpTransport) SendMessage(msg raftpb.Message) {
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

func (t *HttpTransport) getPeerURL(id uint64) string {
	for peer, peerID := range t.peerMap {
		if peerID == id {
			return peer
		}
	}
	return ""
}

func (t *HttpTransport) Receive(w http.ResponseWriter, r *http.Request) {
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
	t.RecvC <- msg
	w.WriteHeader(http.StatusOK)
}
