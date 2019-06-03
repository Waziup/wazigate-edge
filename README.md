# Waziup Edge Server *for Waziup Gateway*

The edge server provides basic endpoints do retrieve, upload and change device, sensor and actuator data.

You can use REST and MQTT on all endpoints.

![Waziup Structure](./assets/waziup_structure.svg)


# Usage

This project is part of the [Waziup Open Innovation Platform](https://www.waziup.eu/). In most cases you do not want to use this repository without the
waziup platform, so have a look the [**Waziup Gateway**](https://github.com/Waziup/waziup-gateway) at [github.com/Waziup/waziup-gateway](https://github.com/Waziup/waziup-gateway).


# Development

## with go (golang) from source

You can compile this project from source with golang and git.
Grab yourself the go language from [golang.org](https://golang.org/) and the
git command line tools with `apt-get git` or from [git-scm.com/download](https://git-scm.com/download).

Now build the waziup-edge executable:

```bash
git clone https://github.com/Waziup/waziup-edge.git
cd waziup-edge
go build .
```

And run the waziup-edge server with:


```bash
waziup-edge
```

## with docker

If you like to use docker you can use the public waziup docker containers at [the Docker Hub](https://hub.docker.com/u/waziup/).
For development you can build this repo on your own using:

```bash
git clone https://github.com/Waziup/waziup-edge.git
cd waziup-edge
docker build --tag=waziup-edge .
docker run -p 4000:80 waziup-edge
```

# Examples

... with JavaScript and [fetch](https://developer.mozilla.org/en-US/docs/Web/API/Fetch_API/Using_Fetch).

### create a new device

```javascript
var resp = await fetch("/devices", {
    method: "POST",
    headers: {
		'Content-Type': 'application/json'
	},
    body: JSON.stringify({
        // id: "5cde6d034b9f61" // let the server choose an id
        name: "My Device 1",    // readable device name
        sensors: [{             // sensors list:
			id: "6f840f0b1",       // sensor id (hardware id)
			name: "My Sensor 1",   // readable name
        }, {
			id: "df34b9f612",
			name: "My Sensor 2",
        }],
        actuators: [{           // actuators list:
			id: "40f034",
			name: "My Actuator 1",
        }],
    })
});
// the device id will be returned
var deviceId = await resp.json();
alert(`new device.id: ${deviceId}`);
```

Console output will be like:
```
new device.id: 5cde6d034b9f610ff8373bdb
```

### delete a device

```javascript
var deviceId = "5cde6d034b9f610ff8373bdb";
fetch(`/devices/${deviceId}`, {
    method: "DELETE",
});
```

### list all devices

```javascript
var resp = await await fetch("/devices");
var devices = await resp.json();
console.log(devices);
```

### create a new sensor *or actuator*

```javascript
var deviceId = "5cde6d034b9f610ff8373bdb";
await fetch(`/devices/${deviceId}/sensors`, {
    method: "POST",
    headers: {
		'Content-Type': 'application/json'
	},
    body: JSON.stringify({
        id: "0ff8373bd",       // sensor id (hardware id)
        name: "My Sensor 3",   // readable name
    })
});
```

The same goes for actuators. Just replace `sensors` with `actuators`.

### list all sensors *or actuators*

```javascript
var deviceId = "5cde6c194b9f610ff8373bda";
var resp = await await fetch(`/devices/${deviceId}/sensors`);
var sensors = await resp.json();
console.log(sensors);
```

### delete a sensor *or actuator*

```javascript
var sensorId = "0ff8373bd";
var deviceId = "5cde6d034b9f610ff8373bdb";
fetch(`/devices/${deviceId}/sensors/${sensorId}`, {
    method: "DELETE",
});
```

### upload a sensor *or actuator* value

```javascript
var sensorId = "0ff8373bd";
var deviceId = "5cde6d034b9f610ff8373bdb";

var value = 42; // numeric value
// or
var value = "Temp45%23"; // string value
// or
var value = {lat: 52, long: 7}; // complex value
// or
var value = {
  time: new Date(),  // value at specific time
  value: 42          // value
};

fetch(`/devices/${deviceId}/sensors/${sensorId}/value`, {
  method: "POST",
  headers: {
    'Content-Type': 'application/json'
  },
  body: JSON.stringify(value)
});
```




### upload multiple sensor *or actuator* values

```javascript
var sensorId = "0ff8373bd";
var deviceId = "5cde6d034b9f610ff8373bdb";

var values = [42, 45, 47]; // values array
// or
var values = ["a", 0x0b, "cde"]; // ...
// or
var values = [
  { // values with timestamp
    time: new Date(),
    value: 42,
  }, {
    time: new Date(),
    value: 43,
  }
]

fetch(`/devices/${deviceId}/sensors/${sensorId}/values`, {
  method: "POST",
  headers: {
    'Content-Type': 'application/json'
  },
  body: JSON.stringify(values)
});
```

### get the last sensor *or actuator* value

```javascript
var sensorId = "0ff8373bd";
var deviceId = "5cde6d034b9f610ff8373bdb";
var resp = await await fetch(`/devices/${deviceId}/sensors/${sensorId}/value`);
var value = await resp.json();
console.log(value);
```

### get multiple sensor *or actuator* values

```javascript
var sensorId = "0ff8373bd";
var deviceId = "5cde6d034b9f610ff8373bdb";
var resp = await await fetch(`/devices/${deviceId}/sensors/${sensorId}/values`);
var values = await resp.json();
console.log(values);
```

### add a Waziup Cloud for synchronization

```javascript
fetch(`/clouds`, {
  method: "POST",
  headers: {
    'Content-Type': 'application/json'
  },
  body: JSON.stringify({
    url: "api.waziup.io:1883", // mqtt port must be included
    paused: true, // default false
    credentials: {
      username: "myUsername",
      token: "myPassword"
    }
  })
});
// the cloud id will be returned
var cloudId = await resp.json();
alert(`new cloud.id: ${cloudId}`);
```

Console output will be like:
```
new cloud.id: 5ce2793d4b9f612a04a7951d
```

### list all configured clouds

```javascript
var resp = await await fetch("/clouds");
var clouds = await resp.json();
console.log(clouds);
```

Output will be like:
```json
{
  "5ce2793d4b9f612a04a7951d": {
    "id": "5ce2793d4b9f612a04a7951d",
    "paused": true,
    "url": "api.waziup.io/api/v2",
    "credentials": {
      "username": "myUsername",
      "token": "myPassword"
    }
  }
}
```

### pause & resume cloud synchronization

```javascript
var cloudId = "5ce2793d4b9f612a04a7951d";
fetch(`/clouds/${cloudId}/paused`, {
  method: "POST",
  headers: {
    'Content-Type': 'application/json'
  },
  body: JSON.stringify(false) // or true
});
```

### change cloud credentials (username and password)

```javascript
var cloudId = "5ce2793d4b9f612a04a7951d";
fetch(`/clouds/${cloudId}/credentials`, {
  method: "POST",
  headers: {
    'Content-Type': 'application/json'
  },
  body: JSON.stringify({
    username: "myNewUserName",
    token: "myNewPassword",
  })
});
```

### change the cloud url

```javascript
var cloudId = "5ce2793d4b9f612a04a7951d";
fetch(`/clouds/${cloudId}/url`, {
  method: "POST",
  headers: {
    'Content-Type': 'application/json'
  },
  body: JSON.stringify("waziup.myserver.com")
});
```

# Use MQTT!

With MQTT you can publish values for the sensor, which is more efficient than the REST interface.
You will need a MQTT client like [Eclipse Mosquitto](https://mosquitto.org/man/mosquitto_sub-1.html) (commandline) or
[HiveMQ MQTT Websocket Client](http://www.hivemq.com/demos/websocket-client/). For the following examples we will use the HiveMQ Client.

Subscriptions are especially usefull if you want to listen for changes and want to get notified when new sensor values arrive. They will be triggered by both Publishes (via MQTT) and Post (via REST).

Connect to your Waziup Gateway using the connection settings:


* Host: Gateway IP
* Port: 80 (for in-browser MQTT via Websocket) or default 1883
* Client: (any)
* MQTT Version: 3.1

You can now publish and subscribe topics like sensor-values or actuator-values.

To make a new subscription, click "Add New Topic Subscription" and enter a valid
topic, like the values-topic from an **existing sensor**. If you have completed the previous examples, you could use `devices/5cd92df34b9f6126f840f0b1/sensors/6f840f0b1/value`.

Remember that messages must be valid JSON, so enquote strings like `"my string"`.


![Hive MQTT Websocket Client](assets/hive_mqtt.png)
See http://www.hivemq.com/demos/websocket-client/.

Equivalent mosquitto calls looks like:

```bash
# Publish Values
mosquitto_pub \
  -t "devices/5cd92df34b9f6126f840f0b1/sensors/df34b9f612/value" \
  -V "mqttv31" \
  -m 456

# Subscribe to topics:
mosquitto_sub \
  -t "devices/5cd92df34b9f6126f840f0b1/sensors/df34b9f612/value" \
  -V "mqttv31"
```

Mosquitto is available for both Linux and Windows.

There are a few things to keep in mind when using MQTT:

* Use port 80 when using MQTT via REST in your browser and port 1883 otherwise. Most clients will have these ports already configured.
* Topics are equivalent to URLs from the REST API. You can use any url as topic and vice versa.
* Subscriptions will be triggered by both Publishes (via MQTT) and Post (via REST).
* Topics do not start with a slash '/'.
* Use only valid JSON objects as payload!

# Configuration

**Env Variables**

```
WAZIUP_HTTP_ADDR  = :80      HTTP Listen Address
WAZIUP_HTTPS_ADDR = :443     HTTPS Listen Address
WAZIUP_MQTT_ADDR  = :1883    MQTT Listen Address
WAZIUP_MQTTS_ADDR = :8883    MQTTS Listen Address

WAZIUP_TLS_CRT =             TLS Cert File (.crt)
WAZIUP_TLS_KEY =             TLS Key File (.key)

WAZIUP_MONGO = localhost:27017     MongoDB Address

WAZIUP_CLOUDS_FILE = clouds.json    Clouds Config File
```

Note that MQTT via Websocket is available together with the REST API on HTTP and HTTPS. To disable serving static files of *www*, use -www "" (an empty string).

**Commandline Arguments**

```
-crt     TLS Cert File (.crt)
-key     TLS Key File (.key)
-www     HTTP files root, default "/var/www"
-db      MongoDB address, default "localhost:27017"
```
Commandline arguments override env variables!
Secure connections will only be used if -crt and -key are present.

**Config Files**

```
clouds.json   Saves /clouds setttings
```
