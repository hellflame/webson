# Webson

[![GoDoc](https://godoc.org/github.com/hellflame/webson?status.svg)](https://godoc.org/github.com/hellflame/webson) [![Go Report Card](https://goreportcard.com/badge/github.com/hellflame/webson)](https://goreportcard.com/report/github.com/hellflame/webson)

Webson is a event-driven websocket-compatible develepment kit for Golang. It's read `/'websən/` , meaning `Websocket-is-On`, `Websocket-and-so-On`. 

Webson is not just a websocket sdk, but a __flexable__, __light-weight__, __event-driven__, __frame-based__ tcp development framework for golang. It's compatible with [websocket RFC6455](https://datatracker.ietf.org/doc/html/rfc6455) prividing client & server APIs, and it's upgraded to handle more complex scenarios among pure server side network applications.

## Aim

The Aim of this package is to help programers build event-driven websocket services and explore more possibilities with websocket.

## Senarios

- [x] Normal websocket client & server
- [x] PRC tunnel
- [x] Message Queue (Zero MQ)
- [x] Customizable TCP based protocol (Authorization & Data Transmission encryption)

## Features

- [x] Event-Driven Style Development
- [x] HTTP Header Control
- [x] Customized Data & Control Type
- [x] Compression Support
- [x] Streaming Support
- [x] Sync & Async Message Processing
- [x] Private Negotiate Control
- [x] Private Data Masking
- [x] Auto Reconnect Client
- [x] Client & Server Connection Integration
- [x] Connection Pool Manage

## Install

```bash
go get -u github.com/hellflame/webson
```

> no any third-party dependence needed

## Quick Start

__Server__:

```go
func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
    
    ws, e := webson.TakeOver(w, r, nil)
    if e != nil { return }
    ws.OnMessage(webson.TextMessage, func(m *webson.Message, a webson.Adapter) {
      msg, _ := m.Read()
      a.Dispatch(webson.TextMessage, append([]byte("recv: "), msg...))
    })
    ws.Start() // don't forget to Start
  })
  
  if e := http.ListenAndServe("127.0.0.1:8000", nil); e != nil {
    panic(e)
  }
}
```

__Client__:

```go
func main() {
  ws, e := webson.Dial("127.0.0.1:8000", nil)
  if e != nil {
    panic(e)
  }
  ws.OnReady(func(a webson.Adapter) {
    a.Dispatch(webson.TextMessage, []byte("hello"))
  })
  ws.OnMessage(webson.TextMessage, func(m *webson.Message, a webson.Adapter) {
    msg, _ := m.Read()
    fmt.Println("from server:", string(msg))
  })

  if e := ws.Start(); e != nil {
    panic(e)
  }
}
```

Run the server code `First` and keep it running, then run the client, you will see the server response to `hello`, and the client will keep silent in the end. `Ctrl` + `C` to stop client or server.

This Client & Server pair shows the basic websocket development example. There are a few __Points__ you may noticed:

1. Server side is a simple `HTTP` service, `r *http.Request` is for negotiate,  `http.ResponseWriter` is for getting `TCP` connection from `HTTP layer`.
2. Server side use `TakeOver` to negotiate with client request, try to `Upgrade` connection to websocket.
3. Client side use `Dial` to connect & negotiate with websocket server.
4. Both Client & Server need `Start` to trigger every event.
5. `Start` will block until connection is closed.
6. When specific `MessageType` is received, the bond method will be executed. `Read` the message bytes from `m *webson.Message`.
7. Send message with `Dispatch`. Using `a webson.Adapter` or `ws` is actually the same, `webson.Adapter` in the bond method is for more structured developing, you should use it more.
8. `ws.OnReady` is a status event watching. There are more status to watch using `ws.OnStatus` , `OnReady` may be used more often, indicating the connection is ready to read & write. It's actually a short cut for `ws.OnStatus(webson.StatusReady, ...)`.

Before detailed _API Reference_, there are few more __cautions__ about the example.

1. Normally, you can only bind one method to a type a message or status, using `OnMessage` or `OnStatus` , it takes a little more to bind more than one method (talk about later). Which means, two `ws.OnMessage(webson.TextMessage, ...)` only the later one can be executed.
2. The client dialed `127.0.0.1:8000` , actually the full address is `ws://127.0.0.1:8000/`. The default schema `ws` and path `/` is omitted.
3. Client & Server will `Ping` the other side by default, and the other side will respond a `Pong` . If you use `ws.OnMessage(webson.PingMessage, ...)` , you need to `Pong()` in there. `PongMessage` also has default handler, treat it with caution, we will talk about it later.

## Use Cases

Some more cases for *Quick Reference*.

### 1. Echo Service

Server simply echo back client input. This is another most basic use of `webson`.

*Client*:

```go
func main() {
	ws, _ := webson.Dial("127.0.0.1:8000/echo", nil)
  
	ws.OnMessage(webson.TextMessage, collectAndEcho)

	ws.Start()
}

func collectAndEcho(m *webson.Message, a webson.Adapter) {
	msg, _ := m.Read()
	fmt.Println("from server:", string(msg))
  
	var input string
	fmt.Scanln(&input)
	if input == "" { return }
	a.Dispatch(webson.TextMessage, []byte(input))
}
```

*Server*:

```go
func main() {
  http.HandleFunc("/echo", func(w http.ResponseWriter, r *http.Request) {
    ws, _ := webson.TakeOver(w, r, nil)
    
    ws.OnMessage(webson.TextMessage, func(m *webson.Message, a webson.Adapter) {
      msg, _ := m.Read()
      a.Dispatch(webson.TextMessage, append([]byte("echo of "), msg...))
    })
    
    ws.Start() // don't forget to Start
  })
  http.ListenAndServe("127.0.0.1:8000", nil)
}
```

`go run server.go` to start the service first, then `go run client.go` to input some thing to interact with server.

[example](examples/simple-echo)

### 2. Synchronized Message Trigging

When you want to asure the message order, you need to make other side in __Synchronize__ mode, setting `webson.Config{Synchronize: true}`.

*Client*:

```go
ws, _ := webson.Dial("127.0.0.1:8000/sync", &webson.DialConfig{
  Config: webson.Config{Synchronize: true},
})

ws.OnReady(func(a webson.Adapter) {
  for i := 0; i <= 10; i++ {
    a.Dispatch(webson.TextMessage, []byte(strconv.Itoa(i)))
  }
  a.Close()
})
ws.OnMessage(webson.TextMessage, func(m *webson.Message, a webson.Adapter) {
  msg, _ := m.Read()
  fmt.Println(string(msg))
})

ws.Start() // don't forget to Start
```

*Server*:

```go
ws, _ := webson.TakeOver(w, r, &webson.Config{Synchronize: true})

// on status is still triggered Asynchronously
ws.OnStatus(webson.StatusClosed, func(s webson.Status, a webson.Adapter) {
  fmt.Println("connection is closed")
})

ws.OnMessage(webson.TextMessage, func(m *webson.Message, a webson.Adapter) {
  msg, _ := m.Read()
  a.Dispatch(webson.TextMessage, append([]byte("recv: "), msg...))
})

ws.Start() // don't forget to Start
```

After starting server `go run server.go`, run the client using `go run client.go` , you will get output then the client will exit:

```bash
recv: 0
recv: 1
recv: 2
recv: 3
recv: 4
recv: 5
recv: 6
recv: 7
recv: 8
recv: 9
recv: 10
```

Both side need to set `Synchronize = true` in order to get the output above. 

> Use It In Caution

Be aware that __Status__ handler is still triggered asynchronously. And the message handler may __block__ the whole receiving progress in the mode, even __Pause__ the whole receiving if you use a *dead loop* in one event handler.

[example](examples/synchronized)

### 3. <span id="eg-large-entity">Large Entity Transmission</span>

When you want to transmit a large message, like a *very large file*, you don't want the other side to react only after receiving the complete data, or it can be processed piece by piece. You can set `webson.Config{TriggerOnStart: true}` , so that the message handler can be triggered once this side receive the first fragment of the message.

*Recommended Server*:

```go
ws, _ := webson.TakeOver(w, r, &webson.Config{TriggerOnStart: true})

// use ReadIter to process msg chunk by chunk
ws.OnMessage(webson.BinaryMessage, func(m *webson.Message, a webson.Adapter) {
  save, _ := os.Create("random-readiter.bin")
  defer save.Close()

  for chunk := range m.ReadIter(2) {
    save.Write(chunk)
  }
})

ws.Start() // don't forget to Start
```

Or, you can choose *Another Read Server*, instead of `ReadIter`:

```go
ws, _ := webson.TakeOver(w, r, &webson.Config{TriggerOnStart: true})

ws.OnMessage(webson.BinaryMessage, func(m *webson.Message, a webson.Adapter) {
  // be careful, TriggerOnStart only means trigger on start, you need to:
  // loop read unfinished msg, until it's complete
  for {
    msg, e := m.Read()
    if e != nil {
      switch e.(type) {
        case webson.MsgYetComplete:
          fmt.Println("waiting for completing msg....")
          time.Sleep(time.Second)
          continue
        default:
        	panic(e)
      }
    }

    save, _ := os.Create("random-read.bin")
    save.Write(msg)
    save.Close()

    break // break out the loop after msg is complete
  }
})

ws.Start() // don't forget to Start
```

*Client*:

```go
ws, _ := webson.Dial("127.0.0.1:8000/large-entity", nil)

ws.OnReady(func(a webson.Adapter) {
  
  buffer := bytes.NewBuffer(nil)
  vessel := make([]byte, 1024)
  h := sha1.New()
  for i := 0; i < 6; i++ {
    rand.Read(vessel)
    h.Write(vessel)
    buffer.Write(vessel)
  }
  fmt.Println("send sha1", hex.EncodeToString(h.Sum(nil))) // checksum the message
  a.DispatchReader(webson.BinaryMessage, buffer)
  
  a.Close()
})

ws.Start() // don't forget to Start
```

The *Recommended Server* uses `ReadIter` to read message, you can consume the message chunk by chunk, so the message won't burst out, you can send `10G` data to your `4G` Ram Server now. `ReadIter` will return a `<-chan []byte` to `range` over, when the message is complete, the channel will be closed, so the `range` loop will come to an end. `ReadIter` accept the parameter to decide the channel size, which __can not__ be `0`, because the first fragment of message need to be sent to the channel and returned, `0` size will cause a __dead lock__. The chunk size is decided by the other side as long as it won't reach the chunk size limit of this side, which for both default limit is `4k`.

In the *Normal Read Server*, the normal `Read` is used in a loop too. As a way of notifying the reader if the message is complete, `MsgYetComplete` error can be returned, you need to check the error, and wait until no error is returned. It's not quite *Golang*, but it's a way to process this kind of not very large message. You don't need to send `10G` data to your `4G` Ram Server in one message, right? If the message is a *Control* message or the size is smaller, only one `Read` is enough.

In the *client*,  `DispatchReader` is used to dispatch the message. Instead of `Dispatch`, `DispatchReader` accept a `io.Reader` as input, it's handy to dispatch messages that implemented `io.Reader` . Here in the case, using `Dispatch` won't change the effect, message will be *split* into pieces to send to server, but `DispatchReader` can send a lot more data, even __unlimited__.

[example](examples/large-entity)

### 4. Simple Message Queue

This show case mainly shows a way of using *Pool Manage*. In order to simplify the interaction, different topic listens is running in other clients as `queue.go`, one client connect to `server` to send message to different topics.

Define a msg struct for client & server.

```go
type msg struct {
	Topic   string `json:"topic"`
  Content string `json:"content"`
}
```

*Server*

```go
pool := webson.NewPool(nil)

http.HandleFunc("/topics", func(w http.ResponseWriter, r *http.Request) {
  topicName := r.Header.Get("topic")
  if topicName == "" {
    topicName = "default"
  }
  fmt.Println("topic listener added for", topicName)
  ws, _ := webson.TakeOver(w, r, nil)
  
  ws.OnReady(func(a webson.Adapter) {
    a.Dispatch(webson.TextMessage, []byte("ready"))
  })
  ws.OnStatus(webson.StatusClosed, func(s webson.Status, a webson.Adapter) {
    fmt.Println("one topic listener has left", topicName)
  })

  pool.Add(ws, &webson.NodeConfig{Group: topicName})
})

http.HandleFunc("/client", func(w http.ResponseWriter, r *http.Request) {
  ws, _ := webson.TakeOver(w, r, nil)
  
  ws.OnMessage(webson.TextMessage, func(m *webson.Message, a webson.Adapter) {
    rawClientMsg, _ := m.Read()
    var clientMsg msg
    json.Unmarshal(rawClientMsg, &clientMsg)
    pool.ToGroup(clientMsg.Topic, webson.TextMessage, []byte(clientMsg.Content))
  })

  // don't forget to Start this non-grouped connection
  ws.Start()
})

http.ListenAndServe("127.0.0.1:8000", nil)
```

*Topic Listener*, `queue.go`. You can use `go run queue.go topicname` to decide which topicname to listen.

```go
var topic string
if len(os.Args) > 1 {
  topic = os.Args[1]
}

ws, e := webson.Dial("127.0.0.1:8000/topics", &webson.DialConfig{
  ClientConfig: webson.ClientConfig{ExtraHeaders: map[string]string{"topic": topic}}})

fmt.Println("waiting for messages......")
ws.OnMessage(webson.TextMessage, func(m *webson.Message, a webson.Adapter) {
  msg, _ := m.Read()
  fmt.Println("topic message:", string(msg))
})

ws.Start()
```

*Client*:

```go
ws, e := webson.Dial("ws://127.0.0.1:8000/client", nil)

loopQuest := func(a webson.Adapter) {
  var topic string
  fmt.Println("which topic you want to join?")
  fmt.Scanln(&topic)
  if topic == "" {
    topic = "default"
  }
  fmt.Println("send anything to the topic?")
  var input string
  for {
    fmt.Scanln(&input)
    if input == "" {
      continue
    }
    js, _ := json.Marshal(msg{Topic: topic, Content: input})
    a.Dispatch(webson.TextMessage, js)
  }
}

ws.OnReady(func(a webson.Adapter) {
  loopQuest(a)
})

ws.Start()
```

Here comes the simplest message queue groups, you need to run `server` and use `queue.go` to create multiple topics (at now or any time), now you can send messages to different topics using `client.go`.

Notice that once a `Connection` is added to a `Pool`, `Start` will be taken over by the pool, __don't__ `Start` the connection manually. The client `Connection` in the pool can __reconnect__ to other side according to  `PoolConfig.ClientRetry` & `PoolConfig.RetryInterval`.

And one more thing, `Pool` can take both *client* & *server* connections in control, they are treated equally for most cases. It makes it fairly simple to create a __Distributed Message Queue__ based on this simple example.

[SimpleMQ](examples/message-queue)

[DistributedMQ](examples/message-queue-distributed)

### 5. Websocket-like Protocol

There are cases when you want to make your server only serve your own service, speaking your own language. You can speak your own websocket dialect in multiple ways.

#### i) Header Control

You can send your own *HTTP Headers* to server and *Verify* it there.

The *Client* side __Dial__ with extra headers by setting `DialConfig.ClientConfig.ExtraHeaders` , which is `map[string]string`. The Server side __TakeOver__ with header verify function by setting `Config.HeaderVerify` , it's a `func(h http.Header) bool`, return `true` to let the connection continue, or `false` to break the connection. `Cookie` in headers is quite common for verification, you can do the verification in `Config.HeaderVerify`.

This is the __least adjust__ to the websocket itself.

#### ii) Private Magic Key

When doing websocket __negotiation__ (Connection Upgrade), there is a pair of special headers set to ensure the connection is a *RFC* defined *websocket* connection. The client will set a header named `Sec-Websocket-Key` with a random string(16 bytes with base64 encoded), and the server will use __sha1__ to digest this random string with a protocol pre-defined __magic key__, and return to the client with a header named `Sec-Websocket-Accept`, then the client will do the same __sha1__, see if the result are same, or the client will break the connection. [RFC](https://datatracker.ietf.org/doc/html/rfc6455#section-1.3).

Here you can set your own __magic key__ by setting `Config.MagicKey` at both side with the same value, like a __password__. The *Client* side config should be like `DialConfig.Config.MagicKey`.

#### iii) Private Mask Key

When transmitting message with entities, the entity can be __masked__ with a random masking key (which will be sent to the other side in the same message frame). It's a way for security. [RFC](https://datatracker.ietf.org/doc/html/rfc6455#section-5.3).

But, the masking key is actually in the data frame, it makes it not that secure. And only the *client* messages is required to send masked frame.

`Webson` allows you to make it more secure by setting a __private mask key__ in `Config.PrivateMask` , which need to be a `[]byte` with size of the times of 4. You can see this as another kind of password with the ability the __symmetrically encrypt__ the message. This won't make the masking process any slower.

You can also make the *Server* to send masking message frame by setting `Config.AlwaysMask = true`.

#### iv) Streaming

As this is not quite a *Standard* stream protocol, streaming will surely make you speak a new *dialect*. *Streaming* will allow you to send multiple large messages __concurrently__. You can enable *Streaming* by setting `Config{EnableStreams: true}` to enable streaming protocol. 

Unlike __private mask key__ or __private magic key__, streaming protocol is a __negotiable__ feature.

All the features can be applied when you __Dial__ the server or __TakeOver__ the client, here shows the key step:

*Server*:

```go
ws, e := webson.TakeOver(w, r, &webson.Config{
  AlwaysMask:    true,
  EnableStreams: true,
  MagicKey:      []byte("masking here & there"),
  PrivateMask:   []byte("bytes length is times 4!"),
  HeaderVerify: func(h http.Header) bool {
    if h.Get("Secret") != "secretkey" {
      fmt.Println("Not Secret!!")
      return false
    }
    if h.Get("Authorization") != fmt.Sprintf("Basic %s", base64.StdEncoding.EncodeToString([]byte("user:pass"))) {
      fmt.Println("User & Password Miss Match!!")
      return false
    }
    return true
  },
})
```

*Client*:

```go
ws, e := webson.Dial("ws://user:pass@127.0.0.1:8000", &webson.DialConfig{
  Config: webson.Config{
    AlwaysMask:    true,
    EnableStreams: true,
    MagicKey:      []byte("masking here & there"),
    PrivateMask:   []byte("bytes length is times 4!"),
  },
  ClientConfig: webson.ClientConfig{
    ExtraHeaders: map[string]string{"Secret": "secretkey"},
  },
})
```

[Full Example](examples/private-protocol)

### More Cases

#### i) [Message Multiplexing](examples/multiplexing)

Every Message goes concurrently.

#### ii) [Override Default Message Handler](examples/override-default)

Maybe you don't want to __Pong__ immediately.

#### iii) [Compression](examples/compression)

When the message is compression profitable, turn on the __Compression__. Such as `json` , `xml` messages.

#### iv) [Benchmark](examples/benchmark)

__How Fast__ can one side send & other side can process.

## Component Reference

API reference by Componets

### 1. Event Handle

There are two types of event you can register handler, `status change` & `message received`.

#### i) OnStatus

```go
func (con *Connection) OnStatus(s Status, action func(Status, Adapter))
```

You can bind `action` for a kind of status using __OnStatus__. 

`Status` can be one of `{StatusReady, StatusClosed, StatusTimeout}`, their meaning is presented in [__Connection Life Cycle__](#connection-life-cycle).

When status handler `action` is invoked, the first argument is the __previous status__ before triggering the handler. For example, when you bind handler `OnStatus(StatusReady, func(s Status, a Adapter))`, the first argument `s Status` can be `StatusYetReady` or `StatusTimeout` , so that you can tell it's a fresh start or recovering from timeout. Code may be like:

```go
...
ws.OnStatus(webson.StatusReady, func(prev Status, a Adapter) {
  if prev == webson.StatusYetReady {
    // do something at fresh start
  } else if prev == webson.StatusTimeout {
    // do something after timeout recovering
  }
})
...
```

> `OnReady` is a shortcut for `OnStatus(StatusReady, ...)` with restriction that previous status be only `StatusYetReady`.

The second argument is a `interface` named [Adapter](#adapter), which is mainly used for `Sending Messages`, we'll discuss it in detail later in [__Message Dispatching__](#message-dispatching). It's  actually the instance of `*Connection`, the current connection itself. You can simply use `*Connection` returned from `Dial` or `TakeOver` as the bind method is a closure function.

#### ii) <span id="on-message">OnMessage</span>

```go
func (con *Connection) OnMessage(t MessageType, action func(*Message, Adapter))
```

You can bind `action` for a kind of message using __OnMessage__. 

There are 5 predefined `MessageType` to use, which are 2 data types `{TextMessage, BinaryMessage}`  and 3 control type: `{CloseMessage, PingMessage, PongMessage}` . You can monitor your own message type using your own `MessageType`.

The first argument of the action is a `*Message`, you can simply `Read` from it in most senarios, we will discuss it later in [__Message Reading__](#message-dispatching).

We've met __Adapter__ above, and we will discuss it later in [__Message Dispatching__](#message-dispatching).

#### iii) Apply

Before we introduce the __Apply__, there is an important thing to be noted:

__OnStatus__ & __OnMessage__ saves handler in a `map[key]handler`. Which means one `Status` only has one handler when using __OnStatus__, and one `MessageType` only has one handler when using __OnMessage__. If you bind handler to one `Status` or one `MessageType` multiple times, the later handler will __replace__ the previous one. 

In most cases, this is not a problem, one handler for one status, one handler for one message type, simple and strong. But, there are also cases you want to deal with one status or one message type in different places. 

For example, different programers developed totally different logics in different part of the program, they all want to be invoked when the desired message or status showed up. Right, this is an engineering problem.

__Why__ the handler saving map can't be a map of series of handler as default, Like `map[key][]handler` ? Because `webson` privided the ability to replace default event handler, like `Pong` after `Ping` , `Pong` to `RefreshPongTime` in case of `Timeout`.

After discussing that much, there comes the `Apply`:

```go
func (con *Connection) Apply(h EventHandler)
```

__Apply__ takes different `EventHandler` implementations as input, here you can `Apply` as much as you need, and __Revoke__ it. Inside `webson` the framework, the __Pool__ manage is developed with *Apply*.

### <span id="message-dispatching">2. Message Reading</span>

There are two kinds of *Message Reading*, __Read__ at once or __ReadIter__ from message stream. Here is the [example](#eg-large-entity).

#### i) Read

```go
func (m *Message) Read() ([]byte, error)
```

When the `Config.TriggerOnStart` is not set, *webson* will trigger the [handler](#on-message) when the message is completely received, __Read__ will return the full content.

When the `Config.TriggerOnStart = true` is set, *webson* will trigger the handler once the first fragment reached this side, you __may not__ get the complete msg at once, and you will get an error __MsgYetComplete__, you have to read it multiple times until it's complete. 

#### ii) ReadIter

```go
func (m *Message) ReadIter(chanSize int) <-chan []byte
```

__ReadIter__ will always generate a `<-chan []byte` for you to *iterate*, no matter the `Config.TriggerOnStart`.

*chanSize* will decide the channel size, and it must be __at least 1__, because there are already chunks to send to the channel, *0 channel* will cause dead lock. 

The main usage of this __ReadIter__ is for processing messages chunk by chunk, and respond to the message at first fragment. There are *extra costs* for it comparing to the simple __Read__, but don't hesitate to use it when it's necessary.

### <span id="message-dispatching">3. Message Dispatching</span>

There are two kinds of *Message Dispatching* for different senarios, __Dispatch__ for simple use and __DispatchReader__ for large entity.

Note that `MessageType >= 8` is __Control Type__ , the `payload` can't be longer than __125__ bytes. [RFC-Control Frames](https://datatracker.ietf.org/doc/html/rfc6455#section-5.5)

#### i) Dispatch

```go
func (con *Connection) Dispatch(t MessageType, p []byte) error
```

Choose a `MessageType` and give the `payload` bytes to dispatch message to other side.

Large payload (larger than chunk size) will be split into chunks for sent.

#### ii) DispatchReader

```go
func (con *Connection) DispatchReader(t MessageType, r io.Reader) error
```

When the message is kind of huge or in some *Reader*, you can use __DispatchReader__.

### 4. Pool Manage

When you want to broadcast a message to all or some of the live connections, no matter it's server side or client side, you can create a pool to do that. 

#### i) NewPool

*Don't create a Pool by &Pool{xxx}, there are initial actions in NewPool*

```go
func NewPool(c *PoolConfig) *Pool
```

You can decide max connections with `PoolConfig.Size`, this is useful on server side. If there are clients to add to the pool, use `PoolConfig.ClientRetry` & `PoolConfig.RetryInterval` to control reconnection.

#### ii) Add

```go
func (p *Pool) Add(c *Connection, config *NodeConfig) error
```

`Add` takes one connection into the pool, *no matter* it's client or server side. 

Note that the `*Connection` __should not__ start yet. The connection's life cycle will be take over by the *Pool*. Though, you can still bind handlers to the `*Connection`, they __won't be overrided__ by the *Pool*.

`*NodeConfig` is mainly to decide this connection's `Name` and `Group`.

#### iii) Dispatch

```go
func (p *Pool) Dispatch(t MessageType, payload []byte)
```

Broadcast message to all connections.

#### iv) ToClients

```go
func (p *Pool) ToClients(t MessageType, payload []byte)
```

Broadcast message only to client connections (*from Dial*).

#### v) ToServers

```go
func (p *Pool) ToServers(t MessageType, payload []byte)
```

Broadcast message only to server connections (*from TakeOver*).

#### vi) ToGroup

```go
func (p *Pool) ToGroup(gName string, t MessageType, payload []byte)
```

Broadcast message to specific group.

#### vii) ToPick

```go
func (p *Pool) ToPick(name string, t MessageType, payload []byte) bool
```

Send message to the connection with the given name.

#### viii) Except

```go
func (p *Pool) Except(name string, t MessageType, payload []byte)
```

Broadcast message to the connections except the given name.

#### ix) Wait

```go
func (p *Pool) Wait()
```

`Wait` is useful when this side is all clients, this method will wait until no connection alive. __No Need__ to use this on server side, `http` service will hold the whole connection period.

#### x) Close

```go
(p *Pool) Close()
```

Manually trigger the pool close. Every connection will try to close.

#### xi) CastOut

```go
func (p *Pool) CastOut(c *Connection)
```

Castout a connection from the pool.

```go
func (p *Pool) CastOutByName(name string) bool
```

Castout a connection with the given name, if there's no connection with the name, a `false` will be returned.

#### xii) OnStatus

```go
func (p *Pool) OnStatus(action func(Status, Adapter))
```

A general `OnStatus` handler for all connections, if any connection has status change, this handler will be triggered.

#### xiii) OnMessage

```go
func (p *Pool) OnMessage(action func(*Message, Adapter))
```

A general `OnMessage` handler for all connections, if any connection receives a message, this handler will be triggered.

Noted that the connection's trigger mode will be reserved here. If connections with and without  `TriggerOnStart` comes to the same pool, this `OnMessage` will act the same as they are in the original connections. __Better__ to keep the *trigger mode* the same.

## Interface Reference

### 1. <span id="adapter">Adapter</span>

```go
type Adapter interface {
  // for connection manage
  Close()

  // for message writing
  Ping() error
  Pong() error
  Dispatch(MessageType, []byte) error
  DispatchReader(MessageType, io.Reader) error

  // for heartbeat monitor
  RefreshPongTime()
  KeepPing(int, int)

  // for pool manage
  Name() string
  Group() string
}
```

`Adapter` is mainly a restricted interface for `connection manage`, `message writing`, `heartbeat monitor` & `pool manage`.

`Connection` satisfies the interface.

### 2. <span id="event-handler">EventHandler</span>

```go
type EventHandler interface {
  Name() string // yes, name is necessary, it's ok to return ""
  OnStatus(Status, Adapter)
  OnMessage(*Message, Adapter)
}
```

`EventHandler` is an interface for `Apply` a set of event handlers for the `Connection`.

1. `Name() string` is for event handler index, so you can __revoke__ this set of handlers. If there's only one `EventHandler` to apply, it's ok to return just `""`.
2. `OnStatus(Status, Adapter)` will be triggered at the status change. `Status` will be the current status.
3. `OnMessage(*Message, Adapter)` will be triggered when a message is received.

## Configuration Reference

### 1. Config

```go
type Config struct {
  HeaderVerify func(http.Header) bool // verify http headers when upgrade connections

  EnableStreams  bool // allow streaming for this connection
  MaxStreams     int  // max streams this side can take. little one will be choosed.
  ChunkSize      int  // max fragment payloa size
  BufferSize     int  // buffer size for reading from connection
  MaxPayloadSize int  // single data frame size limit
  TriggerOnStart bool // message trigger on first fragment
  Synchronize    bool // handlers will be triggered on the main goroutine with the Start

  EnableCompress bool // allow compression for this connection
  CompressLevel  int // compress level defined in deflate

  Timeout *Timeout // all timeout configs

  PingInterval int // how often to ping the other side

  MagicKey    []byte // private magic key, default magic key will be used if not set
  PrivateMask []byte // extra masking key
  AlwaysMask  bool   // mask message even this is the server side
}
```

### 2. Timeout

```go
type Timeout struct {
  HandshakeTimeout int // max wait time for upgrading handshakes
  
  PongTimeout  int // max wait time for this side to receive a pong after ping
  CloseTimeout int // max wait time for other side to send Close after this side send a Close
}
```

### 3. ClientConfig

```go
type ClientConfig struct {
  UseTLS    bool        // use TLS connection no matter the what's the url
  TLSConfig *tls.Config // TLS config for this connection
  ExtraHeaders map[string]string // extra http headers sent for upgrading
}
```

### 4. DialConfig

```go
type DialConfig struct {
	Config
	ClientConfig
}
```

`DialConfig` is for client Dial, combined with general webson `Config` & client only `ClientConfig`.

### 5. PoolConfig

```go
type PoolConfig struct {
  Name          string // use for connection apply
  Size          int    // max connections the pool can hold, 0 to be unlimited
  ClientRetry   int    // client retry count
  RetryInterval int    // client retry interval
}
```

### 6. NodeConfig

```go
type NodeConfig struct {
  Name  string // node name, will be a random string if empty
  Group string // node group
}
```

NodeConfig is for node append in a pool

## Life Cycles

### <span id="connection-life-cycle">1. Connection Life Cycle</span>

For *Server* side, connections await as `http` services, once `TakeOver` (websocket negotiate) succeeded, the *TCP* connection is ready to speak websocket. 

For *Client* side, `Dial` will use the bare *TCP* dial (net.Dial or tls.Dial) to connect the service, using `HTTP` protocol to negotiate with websocket service. After receiving *Upgrade* success headers, the *TCP* connection is ready to speak websocket.

With `Start`, the connection will start reading from other side. Any Message will be parsed, if there is specified `Message Handler` , it will be invoked (synchronously or asynchronously).

There are 4 status during the whole life cycle.

1. `StatusYetReady`: Before `Start`, this is the *default status* for a new connection. You can't watch this status in `OnStatus`, because you don't need to do anything yet.
2. `StatusReady`: After `Start`, it means the connection is ready to send & receive messages. This status is the normal status.
3. `StatusClosed`: When the other side send a `CloseMessage` , this side will close the connection. Or there is accident happens (lost connection, invalid message, service shutdown, etc.), `StatusClosed` will be set. You can't send any message here.
4. `StatusTimeout`: When `Pong` is not received in time, it's `StatusTimeout`. It means something may happen to the connection. You can't write message here, and you may need to check the reason. `StatusTimeout` can be recovered when there is a `Pong` received (default action).

In the long life cycle of a connection, the status may change from `StatusReady` to `StatusTimeout` and from `StatusTimeout` to `StatusReady` many times, which means `StatusTimeout` handler can be triggered multiple times, somehow, `StatusReady` is so special, `OnReady` will only be triggerer once at the beginning, and `OnStatus(StatusReady, func(prev Status, a Adapter))` can be triggered for multiple times, but you can tell from `prev ` status if this is changed from `StatusYetReady` or `StatusTimeout`.

###  2. Message Cycle

```
            +                +
            |      Ping      |
         +> +---------------->
         |  <----------------+
         |  |      Pong      |
         |  |                |
  Ping   |  |    MsgChunks   |
Interval |  +----------------> (if TriggerOnStart, OnMessage
         |  |     ......     |  is triggered here)
         |  |    (Finish)    |
         |  +----------------> OnMessage
         |  |                |
         +> +---------------->
 Pong    |  |     Ping       |
Timeout  |  |                |
         +> <----------------+
            |     Pong       |
            v                v

```

There's a `Ping` loop to send `Ping` message at every `Ping Interval`, the other side will respond a `Pong` by default. If the `Pong` is not received within `Pong Timeout`, a `StatusTimeout` will be triggered.

`Ping` loop & `Pong` response is bond to the connection by default, but you can disable *Ping Loop* by setting `Config.PingInterval = -1` and replace `Pong` handler with `OnMessage`.

One side can send many `MsgChunks` to other side, if not `Config.EnableStreams` , these `MsgChunks` will send sequentially in the connection, or else, there will be different `MsgChunks` from multiple messages sent sequentially in the connection.

## About

Webson comes as the base develop framework for a greater project. Websocket protocol suited the senario most, it has 

- [x] Stateful connection
- [x] Browser side support
- [x] Many well-developed implementation in different languages. Which means other languages may develop the same application to communicate with `webson` .
- [x] Protocol extendable (there are 2 more reserved bits without significant extension taking over, 5 control type & 5 data type undefined)

There's just `message multiplexing` not quite defined in websocket in the [websocket RFC6455](https://datatracker.ietf.org/doc/html/rfc6455) . But it provided possibility to achieve that, like [Fragmentation](https://datatracker.ietf.org/doc/html/rfc6455#section-5.4). `Webson` takes the idea of [`Stream Id`](https://www.rfc-editor.org/rfc/rfc7540.html#section-4.1) from `HTTP/2` protocol, makes it possible to multiplexing messages concurrently. 

In fact, I was absessed with `HTTP/2` once, especially the `server push` , but it's after all a browser optimized `http` protocol, server side applications can't communicate with it that handy when in full-duplex senarios.

It's just most popular golang websocket developemnt kits are quite primative, with only one loop reading from connection, saying it's more efficient that way. `Webson` wants to do better.

### Streamable

This is a common websocket fragment packet with `StreamId`

```
0                   1                   2                   3
0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
+-+-+-+-+-------+-+-------------+-------------------------------+
|F|R|R|R| opcode|M| Payload len |    Extended payload length    |
|I|S|S|S|  (4)  |A|     (7)     |             (16/64)           |
|N|V|V|V|       |S|             |   (if payload len==126/127)   |
| |1|2|3|       |K|             |                               |
+-+-+-+-+-------+-+-------------+ - - - - - - - - - - - - - - - +
|     Extended payload length continued, if payload len == 127  |
+ - - - - - - - - - - - - - - - +-------------------------------+
|                               |Masking-key, if MASK set to 1  |
+-------------------------------+-------------------------------+
| Masking-key (continued)       |C|  StreamID  (Payload Data)   |
+-------------------------------- - - - - - - - - - - - - - - - +
:                     Payload Data continued ...                :
+ - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - +
|                     Payload Data continued ...                |
+---------------------------------------------------------------+
```

If the message is streamlized, the first `2 Bytes` of payload will be used for streams

```
|C|  StreamID  (Payload)        |
```

`1 bit` for cancel the stream, the left `15 bit` for `StreamId `. So the max stream id will be `32768`. Once the id is used up, `Error` will occur.

The `StreamId` used up happens when all messages are streaming not ended, which means 32768 streams are sending messages. This usually won't happen. Once the message is finished, `StreamId` will be released for new message. Right, the connection will try to find a free `StreamId`.
