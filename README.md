# cleanup-events

Small utilty for mass deletion of Kubernetes events.

## Installation

### Compile From Source

These instructions assume you have go installed and a `$GOPATH` set. 
Just run

```bash
make install
```

The binary is installed in `$GOPATH/bin`. If this directory is on your `$PATH`, you are done.
Otherwise you have to copy the binary in a directory of your `$PATH`.

## Usage

```
cleanup-events -h

Usage of cleanup-events:
  -burst int
        Kubernetes client Burst (default 50)
  -dry-run
        If true, no changes will be made
  -duration duration
        Duration for the operation (default 1h0m0s)
  -kubeconfig string
        Path to the kubeconfig file. If not specified, KUBECONFIG env variable is used.
  -qps float
        Kubernetes client QPS (default 200)
  -retries int
        Number of retries for Kubernetes client operations (default 2)
```


