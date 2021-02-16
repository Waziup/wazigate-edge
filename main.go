package main

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"flag"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/Waziup/wazigate-edge/clouds"
	"github.com/Waziup/wazigate-edge/edge"
	_ "github.com/Waziup/wazigate-edge/edge/codecs/json"
	_ "github.com/Waziup/wazigate-edge/edge/codecs/xlpp"
	_ "github.com/Waziup/wazigate-edge/edge/codecs/javascript"
	"github.com/Waziup/wazigate-edge/mqtt"
	"github.com/Waziup/wazigate-edge/tools"
	"github.com/globalsign/mgo/bson"
)

var branch string  // set by compiler
var version string // set by compiler

var static http.Handler

func main() {

	// Remove date and time from logs
	log.SetFlags(log.Flags() &^ (log.Ldate | log.Ltime))

	logSettings := os.Getenv("WAZIUP_LOG")
	if strings.Contains(logSettings, "date") {
		log.SetFlags(log.Flags() | log.Ldate)
	}
	if strings.Contains(logSettings, "time") {
		log.SetFlags(log.Flags() | log.Ltime)
	}
	if strings.Contains(logSettings, "utc") {
		log.SetFlags(log.Flags() | log.LUTC)
	}

	if strings.Contains(logSettings, "error") {
		LogLevel = LogLevelErrors
	}
	if strings.Contains(logSettings, "warn") {
		LogLevel = LogLevelWarnings
	}
	if strings.Contains(logSettings, "verb") {
		LogLevel = LogLevelVerbose
	}
	if strings.Contains(logSettings, "debug") {
		LogLevel = LogLevelDebug
	}

	////////////////////

	log.Println("Waziup API Server")
	if branch != "" {
		log.Printf("This is a %q build, version %q.", branch, version)
	}
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

	/*----------------------------*/

	// Creating the default user in db if there is no user.
	edge.MakeDefaultUser()

	// user, _ := edge.FindUserByUsername( "admin")
	// log.Printf("User %q.\n", user)
	// edge.DeleteUser( user.ID)

	/*------------------------*/

	log.Printf("[     ] Local device ID is %q.\n", edge.LocalID())

	initDevice()

	mqttLogger := log.New(&mqttPrefixWriter{}, "[MQTT ] ", 0)
	mqttServer = &MQTTServer{mqtt.NewServer(mqttAuth, mqttLogger, mqtt.LogLevel(LogLevel))}

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

// type Request struct {
// 	*http.Request
// 	body interface{}
// }

// func (req *Request) Context() context.Context {
// 	return context.WithValue(req.Request.Context(), edge.RequestBodyContextKey{}, &req.body)
// }

////////////////////

func Serve(_resp http.ResponseWriter, _req *http.Request) int {
	var err error

	// if strings.HasSuffix(req.RequestURI, "/") {
	// 	req.RequestURI += "index.html"
	// }

	resp := ResponseWriter{_resp, 200}
	var replacedBody interface{}
	ctx := context.WithValue(_req.Context(), tools.RequestBodyContextKey{}, &replacedBody)
	req := _req.WithContext(ctx)

	if static != nil {

		if !strings.HasPrefix(req.RequestURI, "/apps/") &&
			(strings.HasSuffix(req.RequestURI, ".js") ||
				strings.HasSuffix(req.RequestURI, ".json") ||
				strings.HasSuffix(req.RequestURI, ".css") ||
				strings.HasSuffix(req.RequestURI, ".map") ||
				strings.HasSuffix(req.RequestURI, ".png") ||
				strings.HasSuffix(req.RequestURI, ".ico") ||
				strings.HasSuffix(req.RequestURI, ".jpg") ||
				strings.HasSuffix(req.RequestURI, ".svg") ||
				strings.HasSuffix(req.RequestURI, ".woff") ||
				strings.HasSuffix(req.RequestURI, ".woff2") ||
				strings.HasSuffix(req.RequestURI, ".ttf") ||
				strings.HasSuffix(req.RequestURI, ".html") ||
				strings.HasSuffix(req.RequestURI, ".webmanifest") ||
				strings.HasSuffix(req.RequestURI, "/")) {

			static.ServeHTTP(&resp, req)

			log.Printf("[WWW  ] (%s) %d %s \"%s\"\n",
				req.RemoteAddr,
				resp.status,
				req.Method,
				req.RequestURI)
			return 0
		}
	}

	size := 0
	var body []byte

	if req.Method == http.MethodPut || req.Method == http.MethodPost {

		body, err = ioutil.ReadAll(req.Body)
		size = len(body)
		req.Body.Close()
		if err != nil {
			http.Error(_resp, "400 Bad Request", http.StatusBadRequest)
			return 0
		}
		req.Body = &tools.ClosingBuffer{
			Buffer: bytes.NewBuffer(body),
		}

		router.ServeHTTP(&resp, req)

	} else if req.Method == MethodPublish {
		if cbuf, ok := req.Body.(*tools.ClosingBuffer); ok {
			size = cbuf.Len()
			body = cbuf.Bytes()
		} else {
			body, err = ioutil.ReadAll(req.Body)
			if err != nil {
				http.Error(_resp, "400 Bad Request", http.StatusBadRequest)
				return 0
			}
			req.Body = &tools.ClosingBuffer{
				Buffer: bytes.NewBuffer(body),
			}
		}
		req.Method = http.MethodPost
		router.ServeHTTP(&resp, req)
		req.Method = MethodPublish
	} else {

		router.ServeHTTP(&resp, req)
	}

	req.Context()

	log.Printf("[%s] %s %d %s %q s:%d\n",
		req.Header.Get("X-Tag"),
		req.RemoteAddr,
		resp.status,
		req.Method,
		req.RequestURI,
		size)

	if req.Method == MethodPublish || req.Method == http.MethodPut || req.Method == http.MethodPost {
		if resp.status >= 200 && resp.status < 300 {
			msg := mqtt.Message{
				QoS:   0,
				Topic: req.RequestURI[1:],
				Data:  body,
			}
			if replacedBody != nil {
				data, err := json.Marshal(replacedBody)
				if err != nil {
					log.Printf("[ERR  ] Can not marshal replacedBody: %v", err)
					return 0
				}
				msg.Data = data
			}
			return mqttServer.Server.Publish(nil, &msg)
		}
	}
	return 0
}

////////////////////////////////////////////////////////////////////////////////

type mqttLogWriter struct{}

func (w *mqttLogWriter) Write(data []byte) (n int, err error) {

	if mqttServer != nil && len(data) != 0 {
		if data[len(data)-1] == '\n' {
			data = data[:len(data)-1]
		}
		msg := &mqtt.Message{
			QoS:   0,
			Topic: "sys/log",
			Data:  data,
		}
		mqttServer.Server.Publish(nil, msg)
	}
	return len(data), nil
}

type mqttPrefixWriter struct{}

func (w *mqttPrefixWriter) Write(data []byte) (n int, err error) {
	log.Print(string(data))
	return len(data), nil
}

////////////////////////////////////////////////////////////////////////////////

func initDevice() {
	local, err := edge.GetDevice(edge.LocalID())
	if local == nil {
		err = edge.PostDevices(&edge.Device{
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
		log.Printf("[Err  ] Can not open %q: %s", cloudsFile, err.Error())
		return
	}
	err = clouds.ReadCloudConfig(file)
	if err != nil {
		log.Printf("[Err  ] Can not read %q: %s", cloudsFile, err.Error())
		return
	}

	log.Printf("[Up   ] Read %d from %q:", len(clouds.GetClouds()), cloudsFile)
	for id, cloud := range clouds.GetClouds() {
		log.Printf("[Up   ] Cloud %q: %s@%s", id, cloud.Username, cloud.REST)
	}

	////////////

	clouds.SetDownstream(mqttServer)

	clouds.OnStatus(statusCallback)
	clouds.OnEvent(eventCallback)
}

func statusCallback(cloud *clouds.Cloud, ent clouds.Entity, status *clouds.Status) {
	data, _ := json.Marshal(struct {
		Entity clouds.Entity  `json:"entity"`
		Status *clouds.Status `json:"status"`
	}{ent, status})
	mqttServer.Publish(nil, &mqtt.Message{
		Topic: "clouds/" + cloud.ID + "/status",
		Data:  data,
	})
}

func eventCallback(cloud *clouds.Cloud, event clouds.Event) {
	data, _ := json.Marshal(event)
	mqttServer.Publish(nil, &mqtt.Message{
		Topic: "clouds/" + cloud.ID + "/events",
		Data:  data,
	})
}
