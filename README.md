# Standalone Implementation of Factom's P2P Network

# Summary
This package implements a partial gossip network, limited to peer connectivity and delivery of messages. The application is responsible for triggering fanout, rounds, and message repetition detection. 

This is a complete rework of the code that aims to get rid of uncertainty of connections as well as polling strategies and also add support for handling multiple protocol versions. This will open up opportunities to improve network flow through generational increments without breaking backward compatibility.

Goals:
* Fully configurable, independent, isolated instances
* Add handshake process to connections to distinguish and identify nodes
* Ability to handle multiple protocol schemes starting from version 9
* Reduced network packet overhead in scheme version 10
* Simple integration
* Easier to read code with comprehensive documentation

# Motivation

* Peers are not identifiable beyond ip address. Multiple connections from the same node are not differentiated and assigned random numbers
* Peers have inconsistent information (some data is only available after the first transmission)
* Peers depend on the seed file to join the network even across reboots; peers are not persisted
* Connections are polled in a single loop to check for new data causing unnecessary delays
* Connections have a complicated state machine and correlation with a Peer object
* Complicated program flow with a lot of mixed channels, some of which are polled

Tackling some of these would require significant overhaul of the existing code to the point where it seemed like it would be easier to start from scratch. 


# Specification

Since there are sweeping structural changes, the first introduction should be the new structure:

![structure](https://i.imgur.com/dflR0uj.png)

An **Endpoint** is an internet location to which a node can connect to. It consists of an ip address with a port.

A **Peer** is an active TCP connection between two nodes. They are identified by their **Peer Hash**, a combination of endpoint and a self-reported nonce called **NodeID**. There exists one and only one Peer instance per Hash. Each peer has a protocol adapter.

Peers and Endpoints are in an `n:m` relation to each other. One endpoint has multiple peers and each peer has zero or one endpoint.

**Network** is the package's public interface. It has a channel for sending messages, a channel for receiving messages, methods to start/stop the network, and methods for simple peer management via peer hash. **Messages** are a combination of peer hash and payload.

The **Controller** is responsible for management of peers (accepting new connections, dialing, persistence etc) as well as routing messages to the right peer. 

The biggest change is that new connections will now handshake before being accepted. This gives the network far greater control of which connections to accept and which to reject. After a successful handshake (more on that below), all Peers in the system will have a unique but identifiable peer hash as well as a version specific protocol adapter. 

## Protocol

### Handshake

The handshake starts with an already established TCP connection.

1. A deadline is set for both reading and writing according to the configuration
2. Generate a Handshake containing our preferred version, listen port, network id, and node id and send it across the wire
3. Blocking read the first message
4. Verify that we are in the same network
5. Calculate the minimum of both our and their version
6. Check if we can handle that version and initialize the protocol adapter

If any step fails, the handshake will fail. 

For backward compatibility, the Handshake message is in the same format as protocol v9 requests but it uses the type "Handshake". Nodes running the old software will just drop the invalid message without affecting the node's status in any way.

### 9

Protocol 9 is the legacy (Factomd V6.2.2) protocol with the ability to split messages into parts disabled. V9 has the disadvantage of sending unwanted overhead with every message, namely Network, Version, Length, Address, Part info, NodeID, Address, Port. In the old p2p system this was used to post-load information but now has been shifted to the handshake.

Data is serialized via Golang's gob.

### 10

Protocol 10 is the slimmed down version of V9, containing only the Type, CRC32 of the payload, and the payload itself. Data is also serialized via Golang's gob.