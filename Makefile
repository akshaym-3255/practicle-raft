# Directories inside snapshot to clean before starting
SNAPSHOT_DIRS := snapshots/member1 snapshots/member2 snapshots/member3 snapshots/member4 snapshots/member5 persistent-store

# Go application binary name
BINARY := raftnode

# Command to build the application
BUILD_CMD := go build -o $(BINARY)

# Command to run the application
RUN_CMD1 := go run . -id 1 -listen-client-urls http://localhost:2379  -listen-peer-urls http://localhost:2380  -initial-cluster 1=http://localhost:2380,2=http://localhost:2382,3=http://localhost:2384  --snapshot-dir /Users/akshay.mohite/open-source/akshay-raft/snapshots/member1
RUN_CMD2 := go run .  -id 2 -listen-client-urls http://localhost:2381  -listen-peer-urls http://localhost:2382  -initial-cluster 1=http://localhost:2380,2=http://localhost:2382,3=http://localhost:2384  --snapshot-dir /Users/akshay.mohite/open-source/akshay-raft/snapshots/member2
RUN_CMD3 := go run .  -id 3 -listen-client-urls http://localhost:2383   -listen-peer-urls http://localhost:2384  -initial-cluster 1=http://localhost:2380,2=http://localhost:2382,3=http://localhost:2384  --snapshot-dir /Users/akshay.mohite/open-source/akshay-raft/snapshots/member3

# Default target
.PHONY: all
all: clean build create-dirs run

# Clean target to remove directories inside snapshot and logs
.PHONY: clean
clean:
	@echo "Cleaning snapshot and log directories..."
	@rm -rf $(SNAPSHOT_DIRS)


# Create necessary directories after cleaning
.PHONY: create-dirs


create-dirs:
	@echo "Creating necessary directories..."
	@mkdir -p $(SNAPSHOT_DIRS)

# Build target to compile the Go application
.PHONY: build
build:
	@echo "Building the application..."
	$(BUILD_CMD)

# Run target to start the application
.PHONY: run
run:
	@echo "Starting the application..."
	$(RUN_CMD1)

# Target to clean, create directories, build, and run in one go
.PHONY: fresh-start
fresh-start: clean create-dirs build run

# Clean target to remove the application binary
.PHONY: clean-bin
clean-bin:
	@echo "Cleaning the binary..."
	@rm -f $(BINARY)
