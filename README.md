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
    "ID": "913a73b565c8e2c8ed94497580f619397709b8b6",
    "Priority": 0,
    "Vnode": {
      "Host": "127.0.0.1:54322",
      "Id": "93bb6a44e8e8922afe0dbcbba3a1ed3eaa859850",
      "Meta": "http=127.0.0.1:9091"
    }
  },
  {
    "ID": "e68fc90abb1e381e42e99ecad64b6e8ecc5f0e0b",
    "Priority": 1,
    "Vnode": {
      "Host": "127.0.0.1:54323",
      "Id": "2209bd42504b643b3acfd7e6599699c7a53d962d",
      "Meta": "http=127.0.0.1:9092"
    }
  },
  {
    "ID": "3be51e6010738d73983ef4202ba0c3e421b46360",
    "Priority": 2,
    "Vnode": {
      "Host": "127.0.0.1:54321",
      "Id": "591c0a39be4478ce77124239189bab225a73de75",
      "Meta": "http=127.0.0.1:9090"
    }
  }
]
```
The cluster is now running and can be used.  Details on the HTTP API can be found in the [API docs](./gateways/README.md).

### Development

- When using debug mode a significant performance degrade may be seen.

### Roadmap

Coming Soon!!

- Persistence
- Locking mechanisms
- Authentication, Authorization and Access
