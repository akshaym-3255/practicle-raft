package main

import (
	"akshay-raft/httpapi"
	"akshay-raft/kvstore"
	"akshay-raft/raftnode"
	"flag"
	"fmt"
	"log"
	"strconv"
)

func main() {

	log.SetFlags(log.LstdFlags | log.Lshortfile) // Show date, time, and file name with line number
	log.SetPrefix("DEBUG: ")

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
