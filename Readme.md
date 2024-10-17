
go run . -id 1 -listen-client-urls http://localhost:2379  -listen-peer-urls http://localhost:2380  -initial-cluster 1=http://localhost:2380,2=http://localhost:2382,3=http://localhost:2384  --snapshot-dir /Users/akshay.mohite/open-source/akshay-raft/snapshots/member1

go run .  -id 2 -listen-client-urls http://localhost:2381  -listen-peer-urls http://localhost:2382  -initial-cluster 1=http://localhost:2380,2=http://localhost:2382,3=http://localhost:2384  --snapshot-dir /Users/akshay.mohite/open-source/akshay-raft/snapshots/member2

go run .  -id 3 -listen-client-urls http://localhost:2383   -listen-peer-urls http://localhost:2384  -initial-cluster 1=http://localhost:2380,2=http://localhost:2382,3=http://localhost:2384  --snapshot-dir /Users/akshay.mohite/open-source/akshay-raft/snapshots/member3


