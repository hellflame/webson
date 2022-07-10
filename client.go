package webson

import (
	"bufio"
	"crypto/tls"
	"encoding/base64"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type simpleUrl struct {
	hostPort string
	path     string

	userPassword string
	useTLS       bool
}

func (s *simpleUrl) formatRequestBase() string {
	requestLine := fmt.Sprintf("GET %s HTTP/1.1\r\nHost: %s\r\n", s.path, s.hostPort)
	if s.userPassword != "" {
		requestLine += fmt.Sprintf("Authorization: Basic %s\r\n",
			base64.StdEncoding.EncodeToString([]byte(s.userPassword)))
	}
	return requestLine
}

func Dial(addr string, c *DialConfig) (*Connection, error) {
	p := parseUrl(addr)
	if p == nil {
		panic("address is invalid")
	}
	if c == nil {
		c = &DialConfig{}
	}
	c.url = p
	c.UseTLS = p.useTLS || c.UseTLS || c.TLSConfig != nil
	if c.UseTLS {
		c.dialer = func() (net.Conn, error) {
			return tls.Dial("tcp", c.url.hostPort, c.TLSConfig)
		}
	} else {
		c.dialer = func() (net.Conn, error) {
			return net.Dial("tcp", c.url.hostPort)
		}
	}
	con := &Connection{
		config:   &c.Config,
		client:   &c.ClientConfig,
		isClient: true,
	}
	if e := con.config.setup(); e != nil {
		return nil, e
	}
	if raw, negoConfig, e := negotiate(con.client, con.config); e != nil {
		return nil, e
	} else {
		con.rawConnection = raw
		con.negoSet = *negoConfig
	}
	con.prepare()
	return con, nil
}

func negotiate(client *ClientConfig, config *Config) (con net.Conn, nego *negoSet, e error) {

	raw, e := client.dialer()
	if e != nil {
		return
	}
	defer func() {
		if con == nil {
			raw.Close()
		}
	}()
	challengeKey := createChallengeKey()
	request := client.url.formatRequestBase()
	headers := map[string]string{
		"Upgrade":    "websocket",
		"Connection": "Upgrade",

		"Sec-Websocket-Version": "13",
		"Sec-Websocket-Key":     challengeKey,
	}
	if client.ExtraHeaders != nil {
		for k, v := range client.ExtraHeaders {
			headers[k] = v
		}
	}
	if config.EnableCompress {
		headers["Sec-Websocket-Extensions"] = "permessage-deflate; server_no_context_takeover; client_no_context_takeover"
	}
	if config.EnableStreams {
		headers["Webson-Max-Streams"] = strconv.Itoa(config.MaxStreams)
	}
	for k, v := range headers {
		request += k + ":" + v + "\r\n"
	}
	_, e = raw.Write([]byte(request + "\r\n"))
	if e != nil {
		return
	}
	if config.Timeout.HandshakeTimeout > 0 {
		raw.SetReadDeadline(time.Now().Add(time.Duration(config.Timeout.HandshakeTimeout) * time.Second))
	}

	tmp := bufio.NewReader(raw)
	responsed := false
	verify := make(http.Header)
	malformedResponse := errors.New("malformed response")
	for {
		row, tooLong, e := tmp.ReadLine()
		if tooLong {
			return nil, nil, malformedResponse
		}
		if e != nil {
			// EOF means connection is break
			return nil, nil, e
		}
		left := string(row)
		if left == "" {
			break
		}
		if !responsed {
			responsed = true
			if i := strings.Index(left, " "); i > 0 {
				left = left[i+1:]
			} else {
				return nil, nil, malformedResponse
			}
			if i := strings.Index(left, " "); i > 0 {
				code := left[:i]
				reason := left[i+1:]
				if code != "101" || reason != "Switching Protocols" {
					return nil, nil, errors.New("ws not supported")
				}
			}
		} else {
			if i := strings.Index(left, ":"); i > 0 {
				key := strings.Title(strings.ToLower(left[:i]))
				v := strings.Trim(left[i+1:], " ")
				verify.Add(key, v)
			}
		}
	}

	if config.HeaderVerify != nil && !config.HeaderVerify(verify) {
		return nil, nil, errors.New("header verify not passed")
	}

	acceptKey := verify.Get("Sec-Websocket-Accept")
	if acceptKey == "" || magicDigest(challengeKey, config.MagicKey) != acceptKey {
		e = errors.New("invalid accept key")
		return
	}
	nego = &negoSet{}
	if config.EnableCompress &&
		verify.Get("Sec-Websocket-Extensions") == "permessage-deflate; server_no_context_takeover; client_no_context_takeover" {
		nego.compressable = true
		nego.compressLevel = config.CompressLevel
		if nego.compressLevel == 0 {
			nego.compressLevel = DEFAULT_COMPRESS_LEVEL
		}
	}

	serverStreams := verify.Get("Webson-Max-Streams")
	if config.EnableStreams && serverStreams != "" {
		serverWant, e := strconv.Atoi(serverStreams)
		if e != nil {
			return nil, nil, e
		}
		if serverWant > 0 {
			nego.streamable = true
			nego.maxStreams = config.MaxStreams
			if serverWant < nego.maxStreams {
				nego.maxStreams = serverWant
			}
		}
	}
	return raw, nego, nil
}

func parseUrl(url string) *simpleUrl {
	if i := strings.Index(url, "#"); i > 0 {
		url = url[:i]
	}
	useTLS := false
	if i := strings.Index(url, "://"); i > 0 {
		if url[:i] == "wss" {
			useTLS = true
		}
		url = url[i+3:]
	}
	userPassword := ""
	if i := strings.Index(url, "@"); i > 0 {
		userPassword = url[:i]
		url = url[i+1:]
	}

	hostPort := ""
	path := ""
	if i := strings.Index(url, "/"); i > 0 {
		hostPort = url[:i]
		path = url[i:]
		for _, c := range path {
			if c < ' ' || c == 0x7f {
				return nil
			}
		}
	} else {
		hostPort, url = url, ""
		path = "/"
	}
	if i := strings.Index(hostPort, ":"); i < 0 {
		if useTLS {
			hostPort += ":443"
		} else {
			hostPort += ":80"
		}
	}
	return &simpleUrl{
		hostPort: hostPort,
		path:     path,

		userPassword: userPassword,
		useTLS:       useTLS,
	}
}
