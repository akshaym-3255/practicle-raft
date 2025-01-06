SNAPSHOT_DIRS := snapshots/member1 snapshots/member2 snapshots/member3 snapshots/member4 snapshots/member5 persistent-store

clean:
	@echo "Cleaning snapshot and log directories..."
	@rm -rf $(SNAPSHOT_DIRS)

create-dirs:
	@echo "Creating necessary directories..."
	@mkdir -p $(SNAPSHOT_DIRS)



server1:
	go run .  -id  1 -listen-client-urls http://localhost:2379  -listen-peer-urls http://localhost:2380  -initial-cluster 1=http://localhost:2380,2=http://localhost:2382,3=http://localhost:2384  --snapshot-dir /Users/akshay.mohite/open-source/akshay-raft/snapshots/member1

server2:
	go run .  -id 2  -listen-client-urls http://localhost:2381  -listen-peer-urls http://localhost:2382  -initial-cluster 1=http://localhost:2380,2=http://localhost:2382,3=http://localhost:2384  --snapshot-dir /Users/akshay.mohite/open-source/akshay-raft/snapshots/member2

server3:
	go run .  -id 3  -listen-client-urls http://localhost:2383  -listen-peer-urls http://localhost:2384  -initial-cluster 1=http://localhost:2380,2=http://localhost:2382,3=http://localhost:2384  --snapshot-dir /Users/akshay.mohite/open-source/akshay-raft/snapshots/member3 -join true

server4:
	go run .  -id 4  -listen-client-urls http://localhost:2385  -listen-peer-urls http://localhost:2386  -initial-cluster 1=http://localhost:2380,2=http://localhost:2382,3=http://localhost:2384,4=http://localhost:2386  --snapshot-dir /Users/akshay.mohite/open-source/akshay-raft/snapshots/member4 -join true

server5:
	go run .  -id 5  -listen-client-urls http://localhost:2387  -listen-peer-urls http://localhost:2388  -initial-cluster 1=http://localhost:2380,2=http://localhost:2382,3=http://localhost:2384,4=http://localhost:2386,5=http://localhost:2388  --snapshot-dir /Users/akshay.mohite/open-source/akshay-raft/snapshots/member5 -join true
