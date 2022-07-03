package webson

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
)

// TakeOver mainly tells whether the client is speaking the same kind of websocket protocol.
func TakeOver(w http.ResponseWriter, r *http.Request, c *Config) (*Connection, error) {
	if r.Method != http.MethodGet {
		sendHTTPError(w, http.StatusMethodNotAllowed)
		return nil, errors.New("method is not GET")
	}
	if !r.ProtoAtLeast(1, 1) {
		sendHTTPError(w, http.StatusBadRequest)
		return nil, errors.New("http version too low")
	}

	header := r.Header
	if !strings.EqualFold(header.Get("Connection"), "upgrade") {
		sendHTTPError(w, http.StatusUpgradeRequired)
		return nil, errors.New("connection is not upgrade")
	}
	if !strings.EqualFold(header.Get("Upgrade"), "websocket") {
		sendHTTPError(w, http.StatusBadRequest)
		return nil, errors.New("upgrade is not websocket")
	}
	if header.Get("Sec-Websocket-Version") < "13" {
		sendHTTPError(w, http.StatusBadRequest)
		return nil, errors.New("websocket version too low")
	}
	negotiateKey := header.Get("Sec-Websocket-Key")
	if len(negotiateKey) < 24 {
		sendHTTPError(w, http.StatusBadRequest)
		return nil, errors.New("websocket key is invalid")
	}

	if c == nil {
		c = &Config{}
	}
	if c.HeaderVerify != nil && !c.HeaderVerify(header) {
		sendHTTPError(w, http.StatusUnauthorized)
		return nil, errors.New("header verify not passed")
	}
	if e := c.setup(); e != nil {
		sendHTTPError(w, http.StatusInternalServerError)
		return nil, e
	}

	verified := make(map[string]string)
	streamable := false
	if c.MaxStreams > 0 {
		// negotiate max support streams, choose the little one
		clientWant := header.Get("Webson-Max-Streams")
		if clientWant != "" {
			clientSuggest, e := strconv.Atoi(clientWant)
			if e != nil || clientSuggest <= 0 {
				sendHTTPError(w, http.StatusBadRequest)
				return nil, errors.New("invalid max streams")
			}
			if clientSuggest < c.MaxStreams {
				c.MaxStreams = clientSuggest
			}
			streamable = true
			verified["Webson-Max-Streams"] = strconv.Itoa(c.MaxStreams)
		}
	}
	compressable := false
	if c.EnableCompress {
		if strings.Contains(strings.ToLower(strings.Join(header.Values("Sec-Websocket-Extensions"), " ")),
			"permessage-deflate") {
			compressable = true
			verified["Sec-Websocket-Extensions"] = "permessage-deflate; server_no_context_takeover; client_no_context_takeover"
			if c.CompressLevel == 0 {
				c.CompressLevel = DEFAULT_COMPRESS_LEVEL
			}
		}
	}

	verified["Upgrade"] = "websocket"
	verified["Connection"] = "Upgrade"
	verified["Sec-Websocket-Accept"] = magicDigest(negotiateKey, c.MagicKey)

	hj, ok := w.(http.Hijacker)
	if !ok {
		sendHTTPError(w, http.StatusNotImplemented)
		return nil, errors.New("wrong writer to hijack")
	}
	wsCon, _, e := hj.Hijack()
	if e != nil {
		sendHTTPError(w, http.StatusInternalServerError)
		return nil, errors.New("failed to hijack the link")
	}
	resp := "HTTP/1.1 101 Switching Protocols\r\n"
	for h, v := range verified {
		resp += h + ": " + v + "\r\n"
	}

	if _, e := wsCon.Write([]byte(resp + "\r\n")); e != nil {
		wsCon.Close()
		return nil, e
	}

	con := &Connection{
		rawConnection: wsCon,

		config: c,
		negoSet: negoSet{
			streamable:   streamable,
			compressable: compressable,
		},
	}
	con.prepare()
	return con, nil
}

func sendHTTPError(w http.ResponseWriter, code int) {
	http.Error(w, http.StatusText(code), code)
}
