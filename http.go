package main

import (
	"bytes"
	"crypto/tls"
	"errors"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/Waziup/waziup-edge/mqtt"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     checkOrigin,
}

func checkOrigin(r *http.Request) bool {
	return true
}

func ServeHTTP(resp http.ResponseWriter, req *http.Request) {

	req.Header.Set("X-Secure", "false")
	req.Header.Set("X-Tag", "HTTP ")
	serveHTTP(resp, req)
}

func ServeHTTPS(resp http.ResponseWriter, req *http.Request) {

	req.Header.Set("X-Secure", "true")
	req.Header.Set("X-Tag", "HTTPS")
	serveHTTP(resp, req)
}

////////////////////

type wsWrapper struct {
	tag    string
	conn   *websocket.Conn
	wc     io.WriteCloser
	head   mqtt.FixedHeader
	buf    *bytes.Buffer
	remain int
}

func (w *wsWrapper) Close() error {
	return w.conn.Close()
}

var nonBinaryMessage = errors.New("Unexpected TEXT message.")

func (w *wsWrapper) Read(p []byte) (n int, err error) {

	if w.buf == nil || w.buf.Len() == 0 {

		messageType, data, err := w.conn.ReadMessage()
		if err != nil {
			log.Printf("[%s] (%s) WebSocket Read Error\n %v", w.tag, w.conn.RemoteAddr().String(), err)
			w.conn.Close()
			return 0, err
		}

		if messageType != websocket.BinaryMessage {
			log.Printf("[%s] (%s) WebSocket Error:\n Unexpected TEXT message.", w.tag, w.conn.RemoteAddr().String())
			w.conn.Close()
			return 0, nonBinaryMessage
		}

		w.buf = bytes.NewBuffer(data)
	}

	return w.buf.Read(p)
}

func (w *wsWrapper) Write(data []byte) (int, error) {

	if w.remain == 0 {
		var err error
		w.wc, err = w.conn.NextWriter(websocket.BinaryMessage)
		if err != nil {
			return 0, err
		}
		buf := bytes.NewBuffer(data)
		w.head.Read(buf)
		// num of bytes read by FixedHeader (= header size) + payload length
		w.remain = (len(data) - buf.Len()) + w.head.Length
	}

	w.remain -= len(data)
	_, err := w.wc.Write(data)
	if w.remain == 0 {
		w.wc.Close()
	}
	return len(data), err
}

////////////////////

func serveHTTP(resp http.ResponseWriter, req *http.Request) {

	if req.Header.Get("Upgrade") != "websocket" {

		resp.Header().Set("Access-Control-Allow-Origin", "*")
		Serve(resp, req) // see main.go
	} else {

		proto := req.Header.Get("Sec-WebSocket-Protocol")
		if proto != "mqttv3.1" {
			http.Error(resp, "Requires WebSocket Protocol Header 'mqttv3.1'.", http.StatusBadRequest)
			return
		}

		responseHeader := make(http.Header)
		responseHeader.Set("Sec-WebSocket-Protocol", "mqttv3.1")

		conn, err := upgrader.Upgrade(resp, req, responseHeader)
		if err != nil {
			log.Printf("[%s] (%s) WebSocket Upgrade Failed\n %v", req.Header.Get("X-Tag"), req.RemoteAddr, err)
			return
		}

		var tag string
		if req.Header.Get("X-Secure") == "true" {
			tag = "WSS  "
		} else {
			tag = "WS   "
		}

		wrapper := &wsWrapper{tag: tag, conn: conn}
		mqtt.ServeConn(wrapper, mqttServer)
	}
}

////////////////////////////////////////////////////////////////////////////////

func ListenAndServeHTTP() {

	srv := &http.Server{
		Handler: http.HandlerFunc(ServeHTTP),
	}

	addr := os.Getenv("WAZIUP_HTTP_ADDR")
	if addr == "" {
		addr = ":80"
	}

	listener, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalln("[HTTP] Error:\n", err)
	}

	log.Printf("[HTTP ] HTTP Server at %q. Use \"http://\".", addr)
	log.Printf("[WS   ] MQTT via WebSocket Server at%q. Use \"ws://\".", addr)

	notifyDeamon()
	err = srv.Serve(listener)
	if err != nil {
		log.Println("[HTTP ] Error:")
		log.Fatalln(err)
	}
}

func ListenAndServeHTTPS(cfg *tls.Config) {

	addr := os.Getenv("WAZIUP_HTTPS_ADDR")
	if addr == "" {
		addr = ":443"
	}

	srv := &http.Server{
		Addr:         addr,
		Handler:      http.HandlerFunc(ServeHTTPS),
		TLSConfig:    cfg,
		ReadTimeout:  time.Minute,
		WriteTimeout: time.Minute,
	}

	listener, err := tls.Listen("tcp", addr, cfg)
	if err != nil {
		log.Fatalln("[HTTPS] Error:\n", err)
	}

	log.Printf("[HTTPS] HTTPS Server at %q. Use \"https://\".", addr)
	log.Printf("[WSS  ] MQTT via WebSocket Server at %q.  Use \"wss://\".", addr)
	go func() {
		err = srv.Serve(listener) // will block
		if err != nil {
			log.Fatalln("[HTTPS] Error:\n", err)
		}
	}()
}
