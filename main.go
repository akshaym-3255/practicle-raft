package main

import (
	"akshay-raft/httpapi"
	"akshay-raft/kvstore"
	"akshay-raft/raftnode"
	"flag"
	"fmt"
	"strconv"
)

func main() {
	id := flag.Uint64("id", 1, "node ID")
	clientListenURLs := flag.String("listen-client-urls", "http://localhost:2379", "client listen URL")
	peerListenURLs := flag.String("listen-peer-urls", "http://localhost:2380", "peer listen URL")
	initialCluster := flag.String("initial-cluster", "", "initial cluster configuration")
	join := flag.Bool("join", false, "join an existing cluster")

	snapshotDir := flag.String("snapshot-dir", "", "snapshot dir")
	logDir := snapshotDir

	//peers := flag.String("peers", "http://localhost:2382,http://localhost:2384", "comma-separated list of peer URLs")

	flag.Parse()

	filePath := "/Users/akshay.mohite/open-source/akshay-raft/persistent-store/data-node" + strconv.FormatUint(*id, 10) + ".json"
	fmt.Println(filePath)
	jsonStore := kvstore.NewJsonStore(filePath)
	kvStore := kvstore.NewKeyValueStore(jsonStore)

	rn := raftnode.NewRaftNode(*id, kvStore, *initialCluster, *snapshotDir, *logDir, *join)

	apiServer := httpapi.ApiServer{rn}
	peerServer := httpapi.PeerServer{rn}
	go apiServer.ServeHTTP(*clientListenURLs)
	go peerServer.ServeHTTP(*peerListenURLs)
	go rn.Run()
	select {}
}

// taks to be done
// orgnaize code seprate out raft node processing
// seprate out trasport and server
// key value store
// add support for json file store
// add support for wal log
// add support for snapshot
// add support for cluster membership
// add support for grpc
// add support for any other key value store

/// hwo snapshot and log works in raft
// snapshot

//how raft works
// how wal

// raft bbolt btree
