# fidias [![Build Status](https://travis-ci.org/hexablock/fidias.svg?branch=master)](https://travis-ci.org/hexablock/fidias)

## Table of Contents

- [Getting Started](#installation)
  - [Installation](#installation)
  - [Start a cluster](#start-a-cluster)
- [Documentation](./gateways/README.md)
  - [HTTP API](./gateways/README.md)
- [Roadmap](#roadmap)

### Installation

### Start a cluster
Start the first node in a terminal:

```shell
$ fidiasd -debug
```

Start 2 or more nodes - each in separate terminals.  Change the addresses to appropriately match your configuration.

```shell
$ fidiasd -cluster-addr 127.0.0.1:54322 -http-addr 127.0.0.1:9091 -join 127.0.0.1:54321
$ fidiasd -cluster-addr 127.0.0.1:54323 -http-addr 127.0.0.1:9092 -join 127.0.0.1:54321
...
```

You should see the peers joining the cluster. To confirm check the status of a node:

```shell
$ curl -v -XGET http://127.0.0.1:9090/v1/status
```
Fidias can now be used.  Details on the HTTP API can be found in the [API docs](./gateways/README.md).

### Roadmap

- Persistence
