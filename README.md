# fidias [![Build Status](https://travis-ci.org/hexablock/fidias.svg?branch=master)](https://travis-ci.org/hexablock/fidias)

## Table of Contents

- [Getting Started](#installation)
  - [Installation](#installation)
  - [Start a cluster](#start-a-cluster)
- [Documentation](./gateways/README.md)
  - [HTTP API](./gateways/README.md)
- [Roadmap](#roadmap)

### Installation

1. Download a pre-compiled binary from the [releases page]((https://github.com/hexablock/fidias/releases)).

2. Extract the binary using `unzip` or `tar`.

3. Move the binary into your `$PATH`.

### Start a cluster
The default cluster requires a minimum of 3 nodes to function though this can be changed in the configuration. Below are the steps to spin up a local test cluster.

**1.** Start the first node in a terminal:

  ```shell
  $ fidiasd -debug -bind-addr 127.0.0.1:54321 -http-addr 127.0.0.1:9090
  ```

**2.** Start 2 or more nodes - each in separate terminals.  Change the addresses to appropriately match your configuration.

  ```shell
  $ fidiasd -bind-addr 127.0.0.1:54322 -http-addr 127.0.0.1:9091 -join 127.0.0.1:54321 -debug
  $ fidiasd -bind-addr 127.0.0.1:54323 -http-addr 127.0.0.1:9092 -join 127.0.0.1:54321 -debug
  ...
  ```

**3.** You should start seeing peers joining the cluster. To confirm the cluster is functional, perform a locate call and ensure it responds with locations.  Here's a sample of what the request and response would look like:

```shell
$ curl -XGET http://127.0.0.1:9090/v1/locate/testkey
[
  {
    "ID": "7af630cfe6d6c180dc56caabc76c36965185eabbd71c0a0b4ef800298d147816",
    "Priority": 0,
    "Index": 0,
    "Vnode": {
      "ID": "7d9b8af0cdd93e56e846a13435453525e6a88568474c53675a6084a3d0ad1886",
      "Host": "172.19.0.7:32100",
      "Meta": {
        "http": "127.0.0.1:7700"
      }
    }
  },
  {
    "ID": "d04b86253c2c16d631ac20011cc18beba6db40112c715f60a44d557ee269cd6b",
    "Priority": 1,
    "Index": 0,
    "Vnode": {
      "ID": "d09c14c4e817d5584cda42d1d60fb389c87067ab712f1a91d9e243ff6696c393",
      "Host": "172.19.0.8:32100",
      "Meta": {
        "http": "127.0.0.1:7703"
      }
    }
  },
  {
    "ID": "25a0db7a91816c2b870175567216e140fc30956681c6b4b5f9a2aad437bf22c0",
    "Priority": 2,
    "Index": 2,
    "Vnode": {
      "ID": "4aef6d0fe81410d02859717b124e52971d28eb3f4b3eba3cf1c4c5f579a3594e",
      "Host": "172.19.0.2:32100",
      "Meta": {
        "http": "127.0.0.1:7705"
      }
    }
  }
]
```
The cluster is now running and can be used.  Details on the HTTP API can be found in the [API docs](./gateways/README.md).

### Development

- When using debug mode a significant performance degrade may be seen.

### Roadmap

- 0.4.0
  - Authorization
- 0.3.0
  - Locking mechanisms
  - Initial authorization framework
- 0.2.0
  - Authentication

### Known Issues

- When using fidias in docker on a Mac with persistent storage, a massive performance hit
is incurred due to the way docker volumes and persistence are managed by docker on a Mac.
This is only pertinent for Macs
