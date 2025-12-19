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
cleanup-events [--kubeconfig <kubeconfig-file>] --before <duration>
```


