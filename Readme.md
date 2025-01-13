## how to run it locally

```shell
git clone https://github.com/akshaym-3255/practicle-raft.git
cd practicle-raft
```
### install dependencies
```shell
make install
```
### create necessary directories
```shell
make create-dirs
```

### start a 3 member cluster
```shell
make run-cluster
```

### stop a cluster 
```shell
make stop-cluster
```

### cleanup 
```shell
make clean
```