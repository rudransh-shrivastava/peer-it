# Peer It - Peer to Peer file sharing [WIP]

Instead of relying solely on a single centralized server for file downloads and uploads, peer-it uses direct peer to peer communication.

### What is a Peer?
A Peer is any computer that engages in sharing of a file.
Peers connect to other Peers to share some parts of the file.
They may request or send file chunks to one another.

### What is a Tracker Server?
A tracker server is a centralized server responsible for connecting the peers to one another.
The tracker server is responsible for knowing which peers are engaged in sharing of a particular file.
A network of peers is called a `swarm`.
It knows their public ip:port and is responsible for facilitating a direct connection between two peers.

### What is IPC ?
IPC stands for Inter Process Communcation
The daemon uses Unix Sockets for communication with the CLI (which is run in a different process)

Features [WIP]:
- [x] A tracker server to keep track of files and peers.
- [x] A background daemon that connects to the tracker and uses IPC for communcation with the CLI
- [x] A custom protocol for files, chunks, messages transfer and peer to peer, peer to tracker communication
- [x] A CLI for downloading and registering new files.
- [ ] NAT Traversal, for direct peer to peer communication
- [ ] A `.p2p` or `.pit` file extension that can store the trackers ip, the files checksums, etc.

This project is work in progress.
