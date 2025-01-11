package main

import (
	"akshay-raft/httpapi"
	"akshay-raft/kvstore"
	"akshay-raft/raftnode"
	"flag"
	"strconv"
)

func main() {
	id := flag.Uint64("id", 1, "node ID")
	clientListenURLs := flag.String("listen-client-urls", "http://localhost:2379", "client listen URL")
	peerListenURLs := flag.String("listen-peer-urls", "http://localhost:2380", "peer listen URL")
	initialCluster := flag.String("initial-cluster", "", "initial cluster configuration")
	join := flag.Bool("join", false, "join an existing cluster")
	dataDir := flag.String("data-dir", "", "snapshot dir")
	keyValueStorePath := flag.String("key-store-dir", "", "key store path")
	logDir := dataDir
	flag.Parse()

	keyValueFile := *keyValueStorePath + "/data-node" + strconv.FormatUint(*id, 10) + ".json"
	jsonStore := kvstore.NewJsonStore(keyValueFile)
	kvStore := kvstore.NewKeyValueStore(jsonStore)

	rn := raftnode.NewRaftNode(*id, kvStore, *initialCluster, *dataDir, *logDir, *join)

	apiServer := httpapi.ApiServer{rn}
	peerServer := httpapi.PeerServer{rn}
	go apiServer.ServeHTTP(*clientListenURLs)
	go peerServer.ServeHTTP(*peerListenURLs)
	go rn.Run()
	select {}
}
