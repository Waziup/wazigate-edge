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
    "url": "api.waziup.io:1883",
    "config": {},
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
