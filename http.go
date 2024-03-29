package main

import (
	"crypto/tls"
	"errors"
	"log"
	"net"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/Waziup/wazigate-edge/api"
	"github.com/Waziup/wazigate-edge/mqtt"
	"github.com/gorilla/websocket"
	"github.com/julienschmidt/httprouter"
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
	req.Header.Set("X-Proto", "http")
	serveHTTP(resp, req)
}

func ServeHTTPS(resp http.ResponseWriter, req *http.Request) {

	req.Header.Set("X-Secure", "true")
	req.Header.Set("X-Tag", "HTTPS")
	req.Header.Set("X-Proto", "https")
	serveHTTP(resp, req)
}

////////////////////

type wsWrapper struct {
	conn    *websocket.Conn
	mutex   sync.Mutex
	version byte
	timeout time.Duration
	heap    []byte
}

var errTextMsg = errors.New("unexpected TEXT message")

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func (w *wsWrapper) Read(p []byte) (n int, err error) {
	if len(w.heap) != 0 {
		l := min(len(w.heap), len(p))
		copy(p[:l], w.heap[:l])
		w.heap = w.heap[l:]
		return l, nil
	}
	messageType, data, err := w.conn.ReadMessage()
	if err != nil {
		w.conn.Close()
		return 0, err
	}
	if messageType != websocket.BinaryMessage {
		w.conn.Close()
		return 0, errTextMsg
	}
	w.heap = data
	return w.Read(p)
}

func (w *wsWrapper) ReadPacket() (pkt mqtt.Packet, err error) {
	if w.timeout != 0 {
		w.conn.SetReadDeadline(time.Now().Add(w.timeout))
	}
	pkt, _, err = mqtt.Read(w, w.version)
	if pkt != nil {
		if connectPkt, ok := pkt.(*mqtt.ConnectPacket); ok {
			w.version = connectPkt.Version
		}
	}
	return
}

func (w *wsWrapper) WritePacket(pkt mqtt.Packet) (err error) {
	w.mutex.Lock()
	defer w.mutex.Unlock()
	writer, err := w.conn.NextWriter(websocket.BinaryMessage)
	if err != nil {
		return err
	}
	_, err = pkt.WriteTo(writer, w.version)
	if err != nil {
		writer.Close()
		return err
	}
	return writer.Close()
}

func (w *wsWrapper) Close() error {
	return w.conn.Close()
}

/*
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
*/

////////////////////

func serveHTTPUpgrade(resp http.ResponseWriter, req *http.Request, params httprouter.Params) {

	proto := req.Header.Get("Sec-WebSocket-Protocol")
	if proto != "mqttv3.1" && proto != "mqtt" {
		http.Error(resp, "Requires WebSocket Protocol Header 'mqttv3.1' or 'mqtt'.", http.StatusBadRequest)
		return
	}

	responseHeader := make(http.Header)
	responseHeader.Set("Sec-WebSocket-Protocol", proto)

	conn, err := upgrader.Upgrade(resp, req, responseHeader)
	if err != nil {
		log.Printf("[%s] (%s) WebSocket Upgrade Failed\n %v", req.Header.Get("X-Tag"), req.RemoteAddr, err)
		return
	}

	wrapper := &wsWrapper{conn: conn}
	mqttServer.Serve(wrapper)
}

func serveHTTP(resp http.ResponseWriter, req *http.Request) {

	if req.Header.Get("Upgrade") != "websocket" {

		resp.Header().Set("Access-Control-Allow-Origin", "*")
		Serve(resp, req) // see main.go
	} else {

		api.IsAuthorized(serveHTTPUpgrade, true)(resp, req, nil)
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
	log.Printf("[WS   ] MQTT via WebSocket Server at %q. Use \"ws://\".", addr)

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
