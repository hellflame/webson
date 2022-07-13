# Distributed Message Queue

This example takes an easy way of distribution, the __star__ distribution. 

## Distribute Structure Diagram

```
                      +---------+
                      |OtherNode|         ......
                      |   8001  |
                      +---------+
    +---------+            |             +---------+
    |OtherNode|            |             |OtherNode|
    |   8006  |            |             |   8002  |
    +---------+            v             +---------+
              +------>-----------<-------+
                      |StartNode|
                      |   8000  |
              +------>-----------<-------+
    +---------+            ^             +---------+
    |OtherNode|            |             |OtherNode|
    |   8005  |            |             |   8003  |
    +---------+            |             +---------+
                      +---------+
                      |OtherNode|
                      |   8004  |
                      +---------+
```

Different nodes connect to the very __StartNode__ (listening on port `8000`), then you can bind *any topic* to *any node*, they'll print messages which has their interested topics.

## Message Dispatch

#### 1) Start StartNode

```bash
go run node.go
```

This node will listen on port 8000.

#### 2) Start OtherNode

You can start a number of OtherNode, using different ports, like `8001`:

```bash
go run node.go 8001
```

Giving different port to start more OtherNodes.

#### 3) Create Topic Queue

The general arguemnt for `Topic Queue` is like:

```bash
go run queue.go <TopicName=default> <NodePort=8000>
```

eg:

```bash
go run queue.go rabbit 8000  # connect StartNode and listen on topic `rabbit`
go run queue.go wolf 8001  # connect to service on port 8001 and listen on topic `wolf`
go run queue.go rabbit 8001  # connect to service on port 8001 and listen on topic `rabbit`
```

#### 4) Dispatch Message to a Topic

You can connect to any live node to dispatch the message. The client will connect to the StartNode as default, but you can specify OtherNode with an argument of port.

```bash
go run client.go <NodePort=8000>
```

Then you will enter the dispatch mode, you should input a `Topic` first (like *rabbit* or *wolf*), then you can input anything and press enter, the specified topic queue will receive your message.
