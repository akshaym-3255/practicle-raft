package transport

import (
	"bytes"
	"fmt"
	"github.com/akshaym-3255/practicle-raft/logger"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
	"sync"

	"go.etcd.io/raft/v3/raftpb"
)

type HttpTransport struct {
	id uint64
	// peers   []string
	client  *http.Client
	peerMap map[uint64]string
	RecvC   chan raftpb.Message
	mu      sync.Mutex
}

func NewHTTPTransport(id uint64, peers []string) *HttpTransport {
	peerMap := make(map[uint64]string)
	for _, peer := range peers {
		idHost := strings.Split(peer, "=")
		id, _ := strconv.ParseInt(idHost[0], 10, 64)
		peerMap[uint64(id)] = idHost[1]
	}
	return &HttpTransport{
		id:      id,
		client:  &http.Client{},
		peerMap: peerMap,
		RecvC:   make(chan raftpb.Message, 1024),
	}
}

func (t *HttpTransport) GetPeerURL(id uint64) string {
	if url, ok := t.peerMap[id]; ok {
		return url
	}
	return ""
}

func (t *HttpTransport) AddPeer(newNodeID uint64, newPeerURL string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if newPeerURL != "" {
		t.peerMap[newNodeID] = newPeerURL
	}
	logger.Log.Infof("Added new peer: Node ID %d, URL: %s", newNodeID, newPeerURL)
}

func (t *HttpTransport) RemovePeer(nodeId uint64) {
	t.mu.Lock()
	defer t.mu.Unlock()
	delete(t.peerMap, nodeId)
	logger.Log.Infof("Removed new peer: Node ID %d", nodeId)
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
		logger.Log.Warnf("failed to marshal message: %v", err)
		return
	}
	peerURL := t.GetPeerURL(msg.To)
	if peerURL == "" {
		logger.Log.Warnf("failed to find peer URL for node %d", msg.To)
		return
	}
	logger.Log.Debugf("sending message of type from %d to %d of type %s", msg.From, msg.To, msg.Type)
	url := fmt.Sprintf("%s/raft", peerURL)
	resp, err := t.client.Post(url, "application/octet-stream", bytes.NewReader(data))
	if err != nil {
		logger.Log.Warnf("failed to send message to %s: %v", url, err)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		logger.Log.Warnf("failed to send message to %s, status: %v", url, resp.Status)
	}
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
