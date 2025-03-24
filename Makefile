SNAPSHOT_DIRS := data/member1 data/member2 data/member3 data/member4 data/member5 persistent-store applogs

install:
	go mod tidy
clean:
	@echo "Cleaning snapshot and log directories..."
	@rm -rf $(SNAPSHOT_DIRS)

create-dirs:
	@echo "Creating necessary directories..."
	@mkdir -p $(SNAPSHOT_DIRS)

run-cluster:
	@echo "Starting member1, member2, and member3 with logs..."
	@$(MAKE) member1 &
	@$(MAKE) member2 &
	@$(MAKE) member3 &
	@wait

stop-cluster:
	@echo "Stopping member1, member2, and member3..."
	@for pid in $(shell lsof -ti:2379); do kill -9 $$pid 2>/dev/null || true; done
	@for pid in $(shell lsof -ti:2381); do kill -9 $$pid 2>/dev/null || true; done
	@for pid in $(shell lsof -ti:2383); do kill -9 $$pid 2>/dev/null || true; done
	@for pid in $(shell lsof -ti:2385); do kill -9 $$pid 2>/dev/null || true; done
	@for pid in $(shell lsof -ti:2387); do kill -9 $$pid 2>/dev/null || true; done




member1:
	go run .  -id  1 -listen-client-url http://localhost:2379  -listen-peer-url http://localhost:2380  -initial-cluster 1=http://localhost:2380,2=http://localhost:2382,3=http://localhost:2384  --data-dir ./data/member1  --key-store-dir ./persistent-store/ > applogs/member1-app.log 2>&1

member2:
	go run .  -id 2  -listen-client-url http://localhost:2381  -listen-peer-url http://localhost:2382  -initial-cluster 1=http://localhost:2380,2=http://localhost:2382,3=http://localhost:2384  --data-dir ./data/member2  --key-store-dir ./persistent-store/ > applogs/member2-app.log 2>&1

member3:
	go run .  -id 3  -listen-client-url http://localhost:2383  -listen-peer-url http://localhost:2384  -initial-cluster 1=http://localhost:2380,2=http://localhost:2382,3=http://localhost:2384  --data-dir ./data/member3   --key-store-dir ./persistent-store/ > applogs/member3-app.log 2>&1

member4:
	go run .  -id 4  -listen-client-url http://localhost:2385  -listen-peer-url http://localhost:2386  -initial-cluster 1=http://localhost:2380,2=http://localhost:2382,3=http://localhost:2384,4=http://localhost:2386  --data-dir ./data/member4   --key-store-dir ./persistent-store/ --join=true > applogs/member4-app.log 2>&1

member5:
	go run .  -id 5  -listen-client-url http://localhost:2387  -listen-peer-url http://localhost:2388  -initial-cluster 1=http://localhost:2380,2=http://localhost:2382,3=http://localhost:2384,4=http://localhost:2386,5=http://localhost:2388  --data-dir ./data/member5  --key-store-dir ./persistent-store/ --join=true > applogs/member5-app.log 2>&1
