# Peer It - Peer to Peer  File Sharing Network

Peer it is a decentralised peer to peer file sharing network. 


## Table of Contents  

- [Introduction](#introduction)  
- [Terminology](#terminology)  
- [Features](#features)  
- [How Peer-It Works](#how-peer-it-works)  
- [Getting Started](#getting-started)  
- [Usage](#usage)  
- [License](#license)  


## Introduction  

Instead of relying on a centralised server for file downloads, peer-it uses direct peer to peer communication. 
We do this by using a server that knows which peers are sharing which files, essentially building a table, whenever a new peer joins in, we tell it which other peers are sharing the same file.
Now, that peer can directly connect to all of them and asks for different parts of the files.
The peer also helps the community by uploading parts of the files.



## Terminology  

- `Peer`: A peer is any computer that is engages in sharing of a file.
- `Swarm`: A group of peers sharing a file is called a swarm.
- `Tracker`: A tracker or a tracker server is a centralised server that maintains a table of `swarms`. 
- `Daemon`: A daemon is a background process.
- `IPC`: Processes need to communicate with each other in many situations, IPC stands for Inter Process Communication, it is a mechanism that allows for communication between different processes.
- `Protocol`: A set of rules between communicating parties that defines the structure of all messages being transferred.
- `ISP`: Internet Service Providers.
- `IP Address`: IP stands for Internet Protocol its an address that is a unique identifier assigned to a device on a network, allowing it to communicate with other devices over the internet or a local network.
- `NAT`: NAT stands for Network Address Translation is a method used by ISPs to map different local IP Addresses to a single public IP Address. This is explained in detail in [How Peer-It Works](#how-peer-it-works)  below.



## Features  

- A `.p2p` file that can be shared with anyone similar to `.torrent`
- A tracker server for maintaining tables of swarms.
- A custom protocol for files, chunks, messages transfer and peer to peer, peer to tracker communication.
- A background daemon process for communicating with the tracker and connecting to other peers.
- A command line interface for communicating with the daemon and for download/upload of a file.
- A mechanism for traversing the NAT and punching a hole.



## How Peer-It Works  

Since peer-it is a peer to peer file sharing network, it needs to connect to other peers to be able to share files. This would have been very easy if both the peers were on the same network. 

But modern ISPs use something called a NAT, they have to use a NAT because there is only a limited number of IP (**IPv4**) addresses which is **32-bit** (which comes to **2^32** unique addresses = around **4.29 billion**) available, which is nearly not enough for everyone.
To get around this, ISPs use a NAT to map multiple local IP Addresses (the ones which they have the power of creating since its a local network) to a single public IP Address (that are limited).

Does this mean we will run out of IP Addresses one day? No, modern internet uses IPv6 which is **128-bit** (**2^128** = around **340 trillion trillion trillion** unique addreses)

![Network Address Translator Diagram](https://networkencyclopedia.com/wp-content/uploads/2019/09/network-address-translation-nat.gif)

*Source: [Network Encyclopedia](https://networkencyclopedia.com/network-address-translation-nat/)*

Now that is out of the way, how do we actually get through this? To facilitate any communication between two peers behind a NAT we would need to know which public **IP and Port**. However, since these values can change frequently (especially with dynamic NAT), direct peer-to-peer communication becomes challenging.

To overcome this, we use a **STUN** server, a **STUN** stands for **Session Traversal Utilities for NAT**, its a server which we can connect to and it will send our **Public IP and Port** back to us!

![STUN Server Diagram](https://i.imgur.com/4g99ZRI.png)

*Source: [Research Gate](https://www.researchgate.net/figure/Using-a-STUN-server_fig3_341618550)*

So, that's it right, both peers can hit a STUN server and send over their Public IP and Ports to each other through the tracker and establish a connection right? Nope.
ISPs also block any **inbound** connection request for common residential users that comes from an unknown source.
An inbound connection request means a connection request coming `in` through the NAT and to us primarily due to security, and to avoid people running public servers on their network.

We need to find a way to connect both the peers together. But wait, ISPs allow outbound connection requests, and they also allow inbound connection requests from IPs we have already communicated with in the past

Using this knowledge, we should be able to *trick* the NAT into allowing peer to peer communication. What we do is, once both the peers know their Public IP and Ports, they send it to each other, now both of them will try to send network packets to each others Public IP and Ports at the same time. This *tricks* the NAT into allowing this connection as both the parties have communicated with each other in the past!

This process is basically punching a hole in a NAT aka `hole punching`. 
Once a hole is punched. Both peers can communicate DIRECTLY without needing a server in the middle.

This is exactly what `peer-it` does.

This project's peer to peer network basically comprises of three things.

1. A Tracker server which knows which peers are sharing which files, the job of the tracker server is to tell a new peer about other peers that are sharing the file hat peer is interested in. The Tracker server also helps in punching a hole in the NAT.

2. A Daemon, a daemon is a background process which we run on every device (peer). The daemon connects to the tracker server as soon as it boots up, the daemon is also responsible for connecting to other daemons and sharing files.

3. A CLI, a CLI is a command line interface which uses IPC (unix sockets in this project) to communicate with the daemon 

![Peer It Diagram](https://i.imgur.com/XakXH6a.png)

## Getting Started  

To try peer it, you need to 
1. have [go](https://go.dev/doc/install)  installed.
2. use Linux, because the CLI and Daemon use Unix sockets to communicate (I don't use windows so I didn't feel like a need to implement IPC for windows)

Clone the Github repository:
```bash
git clone https://github.com/rudransh-shrivastava/peer-it/
```


## Usage  

some usage text




## License  

a license probably MIT ?
