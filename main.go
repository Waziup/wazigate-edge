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
	"time"

	"github.com/Waziup/wazigate-edge/api"
	"github.com/Waziup/wazigate-edge/mqtt"
	"github.com/Waziup/wazigate-edge/tools"
	"github.com/globalsign/mgo"
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

	for i := 0; i < 5; i++ {
		log.Printf("[DB   ] Dialing MongoDB at %q...\n", *dbAddr)
		db, err := mgo.Dial("mongodb://" + *dbAddr + "/?connect=direct")
		if err != nil {
			log.Println("[DB   ] MongoDB client error:\n", err)
			time.Sleep(time.Second * 2)
			continue
		} else {

			db.SetSafe(&mgo.Safe{})
			api.DBSensorValues = db.DB("waziup").C("sensor_values")
			api.DBActuatorValues = db.DB("waziup").C("actuator_values")
			api.DBDevices = db.DB("waziup").C("devices")
			break
		}
	}

	////////////////////

	log.Printf("[     ] Local device id is %q.\n", api.GetLocalID())

	api.ReadCloudConfig()

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
	wrapper := ResponseWriter{resp, 200}

	if static != nil {
		if strings.HasPrefix(req.RequestURI, "/www/") {
			req.RequestURI = req.RequestURI[4:]
			req.URL.Path = req.URL.Path[4:]
			static.ServeHTTP(&wrapper, req)

			log.Printf("[WWW  ] (%s) %d %s \"/www%s\"\n",
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
