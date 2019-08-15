package main

import (
	"bytes"
	"crypto/tls"
	"flag"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/Waziup/wazigate-edge/clouds"
	"github.com/Waziup/wazigate-edge/edge"
	"github.com/Waziup/wazigate-edge/mqtt"
	"github.com/Waziup/wazigate-edge/tools"
	"github.com/globalsign/mgo/bson"
)

var static http.Handler

func main() {
	// Remove date and time from logs
	log.SetFlags(log.Flags() &^ (log.Ldate | log.Ltime))

	////////////////////

	log.Println("Waziup API Server")
	log.Println("--------------------")

	////////////////////

	logFile, err := os.Create("log/" + bson.NewObjectId().Hex() + ".txt")
	if err != nil {
		log.Println("[ERR  ]", err)
		log.SetOutput(io.MultiWriter(os.Stdout, &mqttLogWriter{}))
	} else {
		defer logFile.Close()
		log.SetOutput(io.MultiWriter(os.Stdout, logFile, &mqttLogWriter{}))
	}

	tlsCertStr := os.Getenv("WAZIUP_TLS_CRT")
	tlsKeyStr := os.Getenv("WAZIUP_TLS_KEY")

	tlsCert := flag.String("crt", tlsCertStr, "TLS Cert File (.crt)")
	tlsKey := flag.String("key", tlsKeyStr, "TLS Key File (.key)")

	wwwStr, ok := os.LookupEnv("WAZIUP_WWW")
	if !ok {
		wwwStr = "/var/www"
	}
	www := flag.String("www", wwwStr, "HTTP files root")

	dbAddrStr, ok := os.LookupEnv("WAZIUP_MONGO")
	if !ok {
		dbAddrStr = "localhost:27017"
	}
	dbAddr := flag.String("db", dbAddrStr, "MongoDB address")

	flag.Parse()

	////////////////////

	if *www != "" {
		log.Printf("[WWW  ] Serving from %q\n", *www)
		static = http.FileServer(http.Dir(*www))
	} else {
		log.Printf("[WWW  ] Not serving www files.\n")
	}

	////////////////////

	log.Printf("[DB   ] Dialing MongoDB at %q...\n", *dbAddr)
	err = edge.Connect("mongodb://" + *dbAddr + "/?connect=direct")
	if err != nil {
		log.Fatalf("[DB   ] MongoDB client error: %v\n", err)
	}

	////////////////////

	log.Printf("[     ] Local device id is %q.\n", edge.LocalID())

	initDevice()

	initSync()

	////////////////////

	if *tlsCert != "" && *tlsKey != "" {

		cert, err := ioutil.ReadFile(*tlsCert)
		if err != nil {
			log.Println("Error reading", *tlsCert)
			log.Fatalln(err)
		}

		key, err := ioutil.ReadFile(*tlsKey)
		if err != nil {
			log.Println("Error reading", *tlsKey)
			log.Fatalln(err)
		}

		pair, err := tls.X509KeyPair(cert, key)
		if err != nil {
			log.Println("TLS/SSL 'X509KeyPair' Error")
			log.Fatalln(err)
		}

		cfg := &tls.Config{Certificates: []tls.Certificate{pair}}

		ListenAndServeHTTPS(cfg)
		ListenAndServeMQTTTLS(cfg)
	}

	ListenAndServerMQTT()
	ListenAndServeHTTP() // will block
}

///////////////////////////////////////////////////////////////////////////////

type ResponseWriter struct {
	http.ResponseWriter
	status int
}

func (resp *ResponseWriter) WriteHeader(statusCode int) {
	resp.status = statusCode
	resp.ResponseWriter.WriteHeader(statusCode)
}

////////////////////

func Serve(resp http.ResponseWriter, req *http.Request) {

	if strings.HasSuffix(req.RequestURI, "/") {
		req.RequestURI += "index.html"
	}

	wrapper := ResponseWriter{resp, 200}

	if static != nil {

		if strings.HasSuffix(req.RequestURI, ".js") ||
			strings.HasSuffix(req.RequestURI, ".css") ||
			strings.HasSuffix(req.RequestURI, ".map") ||
			strings.HasSuffix(req.RequestURI, ".png") ||
			strings.HasSuffix(req.RequestURI, ".svg") ||
			strings.HasSuffix(req.RequestURI, ".html") {

			static.ServeHTTP(&wrapper, req)

			log.Printf("[WWW  ] (%s) %d %s \"%s\"\n",
				req.RemoteAddr,
				wrapper.status,
				req.Method,
				req.RequestURI)
			return
		}
	}

	size := 0

	if req.Method == http.MethodPut || req.Method == http.MethodPost {

		body, err := ioutil.ReadAll(req.Body)
		size = len(body)

		req.Body.Close()
		if err != nil {
			http.Error(resp, "400 Bad Request", http.StatusBadRequest)
			return
		}
		req.Body = &tools.ClosingBuffer{
			Buffer: bytes.NewBuffer(body),
		}
	}

	if req.Method == "PUBLISH" {
		req.Method = http.MethodPost
		router.ServeHTTP(&wrapper, req)
		req.Method = "PUBLISH"
	} else {
		router.ServeHTTP(&wrapper, req)
	}

	log.Printf("[%s] %s %d %s %q s:%d\n",
		req.Header.Get("X-Tag"),
		req.RemoteAddr,
		wrapper.status,
		req.Method,
		req.RequestURI,
		size)

	if cbuf, ok := req.Body.(*tools.ClosingBuffer); ok {
		// log.Printf("[DEBUG] Body: %s\n", cbuf.Bytes())
		msg := &mqtt.Message{
			QoS:   0,
			Topic: req.RequestURI[1:],
			Data:  cbuf.Bytes(),
		}

		// if wrapper.status >= 200 && wrapper.status < 300 {
		if req.Method == http.MethodPut || req.Method == http.MethodPost {
			mqttServer.Publish(nil, msg)
		}
		// }
	}
}

////////////////////////////////////////////////////////////////////////////////

type mqttLogWriter struct{}

func (w *mqttLogWriter) Write(data []byte) (n int, err error) {
	if mqttServer != nil && len(data) != 0 {
		data2 := make([]byte, len(data))
		copy(data2, data)
		if data2[len(data2)-1] == '\n' {
			data2 = data2[:len(data2)-1]
		}
		go func(data []byte) {
			msg := &mqtt.Message{
				QoS:   0,
				Topic: "sys/log",
				Data:  data,
			}
			mqttServer.Publish(nil, msg)
		}(data2)
	}
	return len(data), nil
}

////////////////////////////////////////////////////////////////////////////////

func initDevice() {
	local, err := edge.GetDevice(edge.LocalID())
	if err != nil {
		log.Fatalf("[DB   ] Err %v", err)
	}
	if local == nil {
		err = edge.PostDevice(&edge.Device{
			ID:   edge.LocalID(),
			Name: "Gateway " + edge.LocalID(),
		})
		if err != nil {
			log.Fatalf("[DB   ] Err %v", err)
		}
	}
}

////////////////////////////////////////////////////////////////////////////////

func getCloudsFile() string {
	cloudsFile := os.Getenv("WAZIUP_CLOUDS_FILE")
	if cloudsFile == "" {
		return "clouds.json"
	}
	return cloudsFile
}

func initSync() {
	cloudsFile := getCloudsFile()
	file, err := os.Open(cloudsFile)
	if err != nil {
		log.Printf("[Err  ] Can not read %q: %s", cloudsFile, err.Error())
	}
	err = clouds.ReadCloudConfig(file)
	if err != nil {
		log.Printf("[Err  ] Can not read %q: %s", cloudsFile, err.Error())
	}
}
