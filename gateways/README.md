# HTTP Gateway
The HTTP gateway provides access to the fidias cluster via HTTP API's

## Endpoints
The following are a list of available endpoints:

#### /v1/kv/*{ key }*
This endpoint allows to perform key-value pair operations.

| Method   | Body | Description |
|----------|------|-------------|
| GET      |      | Get a key |
| POST/PUT | Any data as the value | Set a key |
| DELETE   |      | Delete a key |

#### /v1/lookup/*{ key }*
This endpoint performs a lookup on the key returning the max or specified number of successors.

| Method   | Parameters | Description |
|----------|------------|-------------|
| GET      | **n** Number of successors | Lookup key |

#### /v1/locate/*{ key }*
This endpoint finds replica locations for a given key.

| Method   | Parameters | Description |
|----------|------------|-------------|
| GET      | **r** Number of replicas | Locate key replicas |

#### /v1/hexalog/*{ key }*
This endpoint allows operations directly against the log.  Its use is intended for admin purposes only.

#### /v1/blox
This endpoint allow operations on the content-addressable storage

#### /v1/status
This endpoint returns the status of the node.

| Method   | Description |
|----------|-------------|
| GET      | Get the status of the node |
