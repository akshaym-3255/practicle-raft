SNAPSHOT_DIRS := snapshots/member1 snapshots/member2 snapshots/member3 snapshots/member4 snapshots/member5 persistent-store

.PHONY: clean
clean:
	@echo "Cleaning snapshot and log directories..."
	@rm -rf $(SNAPSHOT_DIRS)

.PHONY: create-dirs
create-dirs:
	@echo "Creating necessary directories..."
	@mkdir -p $(SNAPSHOT_DIRS)


.PHONY: server1 server1 server1

server1:
	go run .  -id  1 -listen-client-urls http://localhost:2379  -listen-peer-urls http://localhost:2380  -initial-cluster 1=http://localhost:2380,2=http://localhost:2382,3=http://localhost:2384  --snapshot-dir /Users/akshay.mohite/open-source/akshay-raft/snapshots/member1

server2:
	go run .  -id 2  -listen-client-urls http://localhost:2381  -listen-peer-urls http://localhost:2382  -initial-cluster 1=http://localhost:2380,2=http://localhost:2382,3=http://localhost:2384  --snapshot-dir /Users/akshay.mohite/open-source/akshay-raft/snapshots/member2

server3:
	go run .  -id 3  -listen-client-urls http://localhost:2383  -listen-peer-urls http://localhost:2384  -initial-cluster 1=http://localhost:2380,2=http://localhost:2382,3=http://localhost:2384  --snapshot-dir /Users/akshay.mohite/open-source/akshay-raft/snapshots/member3


