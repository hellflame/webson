# Webson

Webson is a event-driven websocket-compatable develepment kit for Golang. It's read `/'websən/` , meaning `Websocket-is-On`, `Websocket-and-so-On`. 

Webson is not just a websocket sdk, but a flexable, light-weight, event-driven, frame-based tcp development framework for golang. It's compatable with [websocket RFC6455](https://datatracker.ietf.org/doc/html/rfc6455) prividing client & server APIs, and it's upgraded to handle more complex scenarios among pure server side network applications.

## Aim

The Aim of this package is to help programers build event-driven websocket services and explore more possibilities with websocket.

## Senarios

- [x] Normal websocket client & server
- [x] PRC tunnel
- [x] Message Queue (Zero MQ)
- [x] Customizable TCP based protocol (Authorization & Data Transmission encryption)

## Features

- [x] Event-Driven Development
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

#### ii) OnMessage

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

__Apply__ takes different `EventHandler` implementations as input, here you can `Apply` as much as you need, and __Revoke__ it. 

### <span id="message-dispatching">2. Message Reading</span>

### <span id="message-dispatching">3. Message Dispatching</span>



## Interface Reference

### 1. <span id="adapter">Adapter</span>



### 2. <span id="event-handler">EventHandler</span>



## Struct Reference

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

###  2. Data Cycle

## About

Webson comes as the base develop framework for a greater project. Websocket protocol suited the senario most, it has 

- [x] Stateful connection
- [x] Browser side support
- [x] Many well-developed implementation in different languages. Which means other languages may develop the same application to communicate with `webson` .
- [x] Protocol extendable (there are 2 more reserved bits without significant extension taking over, 5 control type & 5 data type undefined)

There's just `message multiplexing` not quite defined in websocket in the [websocket RFC6455](https://datatracker.ietf.org/doc/html/rfc6455) . But it provided possibility to achieve that, like [Fragmentation](https://datatracker.ietf.org/doc/html/rfc6455#section-5.4). `Webson` takes the idea of [`Stream Id`](https://www.rfc-editor.org/rfc/rfc7540.html#section-4.1) from `HTTP/2` protocol, makes it possible to multiplexing messages concurrently. 

In fact, I was absessed with `HTTP/2` once, especially the `server push` , but it's after all a browser optimized `http` protocol, server side applications can't communicate with it that handy when in full-duplex senarios.

It's just most popular golang websocket developemnt kits are quite primative, with only one loop reading from connection, saying it's more efficient that way. `Webson` wants to do better.
