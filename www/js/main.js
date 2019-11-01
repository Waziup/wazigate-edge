(function() {

  /*
  virgin =  $.box
  className: "virgin"
  on: click: () ->
  name = prompt "Please enter device name:", "New Device"
  return if ! name
  device = 
      name: name
      sensors: []
      actuators: []
  resp = await fetch "devices",
      method: "POST"
      body: JSON.stringify device
  if ! resp.ok
      text = await resp.text()
      alert "Error:\n"+text
      return
  id = await resp.text()
  device.id = id
  inflated = inflateDevice device
  content1.$.insertBefore inflated, virgin
  devices.push device
  heading.text "#{devices.length} Devices"
  return        
  , [$.text "Create new Device"]

  content1.append virgin
  */
  var MQTT, breadcrumbs, conncetMQTT, content1, content2, formatTime, heading, inflateActuator, inflateCloud, inflateDevice, inflateLog, inflateSensor, log, logOpener, navigate, refreshUptime, setStatus, showActuator, showBreadcrumbs, showClouds, showDevice, showDevices, showSensor, status, statusbar, subheading1, subheading2, tagRegexp, timeSuffixes, uptimeText, uptimeTicker;

  switch ($.platform()) {
    case "windows":
      $.body.addClass("windows");
      break;
    case "linux":
      $.body.addClass("linux");
      break;
    case "mac":
      $.body.addClass("mac");
  }

  navigate = function(hash) {
    var match;
    if (hash === "") {
      location.hash = "#devices";
      return;
    }
    if (location.hash !== "#" + hash) {
      location.hash = "#" + hash;
      return;
    }
    showBreadcrumbs(hash);
    if (hash === "devices") {
      showDevices();
      return;
    }
    if (hash === "clouds") {
      showClouds();
      return;
    }
    match = hash.match(/^devices\/(\w+)$/);
    if (match !== null) {
      showDevice(match[1]);
      return;
    }
    match = hash.match(/^devices\/(\w+)\/sensors$/);
    if (match !== null) {
      location.hash = `#devices/${match[1]}`;
      return;
    }
    match = hash.match(/^devices\/(\w+)\/actuators$/);
    if (match !== null) {
      location.hash = `#devices/${match[1]}`;
      return;
    }
    match = hash.match(/^devices\/(\w+)\/sensors\/(\w+)$/);
    if (match !== null) {
      showSensor(match[1], match[2]);
      return;
    }
    match = hash.match(/^devices\/(\w+)\/actuators\/(\w+)$/);
    if (match !== null) {
      showActuator(match[1], match[2]);
      return;
    }
  };

  //###############################################################################
  // location.hash = "#devices"
  heading = $("#heading");

  content1 = $("#content1");

  subheading1 = $("#subheading1");

  content2 = $("#content2");

  subheading2 = $("#subheading2");

  showDevices = async function() {
    var device, devices, resp, virgin;
    resp = (await fetch("devices"));
    if (!resp.ok) {
      showRespError(resp);
      return;
    }
    devices = (await resp.json());
    heading.text(`${devices.length} Devices`);
    subheading1.hide();
    subheading2.hide();
    content1.show().text("");
    content2.hide();
    for (device of devices) {
      content1.append(inflateDevice(device));
    }
    virgin = $.box({
      className: "virgin",
      on: {
        click: async function() {
          var id, inflated, name, text;
          name = prompt("Please enter device name:", "New Device");
          if (!name) {
            return;
          }
          device = {
            name: name,
            sensors: [],
            actuators: []
          };
          resp = (await fetch("devices", {
            method: "POST",
            body: JSON.stringify(device)
          }));
          if (!resp.ok) {
            text = (await resp.text());
            alert("Error:\n" + text);
            return;
          }
          id = (await resp.text());
          device.id = id;
          inflated = inflateDevice(device);
          content1.$.insertBefore(inflated, virgin);
          devices.push(device);
          heading.text(`${devices.length} Devices`);
        }
      }
    }, [$.text("Create new Device")]);
    content1.append(virgin);
  };

  inflateDevice = function(device) {
    var inflated, nameText;
    inflated = $.box({
      className: "box"
    }, [
      $.create("img",
      {
        props: {
          src: "img/device.png"
        }
      }),
      $.create("h2",
      {},
      [
        $.create("a",
        {
          props: {
            href: `#devices/${device.id}`
          }
        },
        [nameText = $.text(device.name)])
      ]),
      $.box({
        className: "floating"
      },
      [
        $.create("img",
        {
          props: {
            src: "img/edit.svg",
            title: "Edit Name"
          },
          on: {
            click: async function() {
              var name,
        resp,
        text;
              name = prompt(`Please enter a new name for "${device.name}":`,
        device.name);
              if (name && name !== device.name) {
                resp = (await fetch(`devices/${device.id}/name`,
        {
                  method: "POST",
                  body: name
                }));
                if (!resp.ok) {
                  text = (await resp.text());
                  alert("Error:\n" + text);
                  return;
                }
                nameText.textContent = name;
              }
            }
          }
        }),
        $.create("img",
        {
          props: {
            src: "img/delete.svg",
            title: "Delete"
          },
          on: {
            click: async function() {
              var resp,
        text;
              if (confirm(`Delete "${device.name}"?\nThis will also delete all device data points.`)) {
                resp = (await fetch(`devices/${device.id}`,
        {
                  method: "DELETE"
                }));
                if (!resp.ok) {
                  text = (await resp.text());
                  alert("Error:\n" + text);
                  return;
                }
                $(inflated).remove();
              }
            }
          }
        })
      ]),
      $.create("a",
      {
        className: "id",
        props: {
          href: `#devices/${device.id}`
        }
      },
      [$.text(`ID ${device.id}`)])
    ]);
    return inflated;
  };

  //###################
  showDevice = async function(deviceID) {
    var actuator, device, ref, ref1, resp, sensor, virgin1, virgin2;
    resp = (await fetch(`devices/${deviceID}`));
    if (!resp.ok) {
      showRespError(resp);
      return;
    }
    device = (await resp.json());
    heading.text(device.name);
    content1.show().text("");
    subheading1.show().text(`${device.sensors.length} Sensors`);
    ref = device.sensors;
    for (sensor of ref) {
      content1.append(inflateSensor(deviceID, sensor));
    }
    virgin1 = $.box({
      className: "virgin",
      on: {
        click: async function() {
          var id, inflated, name, text;
          name = prompt("Please enter sensor name:", "New Sensor");
          if (!name) {
            return;
          }
          sensor = {
            name: name,
            value: null,
            time: new Date()
          };
          resp = (await fetch(`devices/${deviceID}/sensors`, {
            method: "POST",
            body: JSON.stringify(sensor)
          }));
          if (!resp.ok) {
            text = (await resp.text());
            alert("Error:\n" + text);
            return;
          }
          id = (await resp.text());
          sensor.id = id;
          inflated = inflateSensor(device.id, sensor);
          content1.$.insertBefore(inflated, virgin1);
        }
      }
    }, [$.text("Create new Sensor")]);
    content1.append(virgin1);
    //###
    content2.show().text("");
    subheading2.show().text(`${device.actuators.length} Actuators`);
    ref1 = device.actuators;
    for (actuator of ref1) {
      content2.append(inflateActuator(deviceID, actuator));
    }
    virgin2 = $.box({
      className: "virgin",
      on: {
        click: async function() {
          var id, inflated, name, text;
          name = prompt("Please enter actuator name:", "New Actuator");
          if (!name) {
            return;
          }
          actuator = {
            name: name,
            value: null,
            time: new Date()
          };
          resp = (await fetch(`devices/${deviceID}/actuators`, {
            method: "POST",
            body: JSON.stringify(actuator)
          }));
          if (!resp.ok) {
            text = (await resp.text());
            alert("Error:\n" + text);
            return;
          }
          id = (await resp.text());
          actuator.id = id;
          inflated = inflateActuator(device.id, actuator);
          content2.$.insertBefore(inflated, virgin2);
        }
      }
    }, [$.text("Create new Actuator")]);
    content2.append(virgin2);
  };

  inflateSensor = function(deviceID, sensor) {
    var inflated, nameText, valueText;
    if (sensor.value === null) {
      valueText = "(none)";
    } else {
      valueText = JSON.stringify(sensor.value, null, 2);
    }
    inflated = $.box({
      className: "box"
    }, [
      $.create("img",
      {
        props: {
          src: "img/sensor.png"
        }
      }),
      $.create("h2",
      {},
      [
        $.create("a",
        {
          props: {
            href: `#devices/${deviceID}/sensors/${sensor.id}`
          }
        },
        [nameText = $.text(sensor.name)])
      ]),
      $.box({
        className: "floating"
      },
      [
        $.create("img",
        {
          props: {
            src: "img/edit.svg",
            title: "Edit Name"
          },
          on: {
            click: async function() {
              var name,
        resp,
        text;
              name = prompt(`Please enter a new name for "${sensor.name}":`,
        sensor.name);
              if (name && name !== sensor.name) {
                resp = (await fetch(`devices/${deviceID}/sensors/${sensor.id}/name`,
        {
                  method: "POST",
                  body: name
                }));
                if (!resp.ok) {
                  text = (await resp.text());
                  alert("Error:\n" + text);
                  return;
                }
                nameText.textContent = name;
              }
            }
          }
        }),
        $.create("img",
        {
          props: {
            src: "img/delete.svg",
            title: "Delete"
          },
          on: {
            click: async function() {
              var resp,
        text;
              if (confirm(`Delete "${sensor.name}"?\nThis will also delete all sensor data points.`)) {
                resp = (await fetch(`devices/${deviceID}/sensors/${sensor.id}`,
        {
                  method: "DELETE"
                }));
                if (!resp.ok) {
                  text = (await resp.text());
                  alert("Error:\n" + text);
                  return;
                }
                $(inflated).remove();
              }
            }
          }
        })
      ]),
      $.box({
        className: "property",
        attr: {
          "data-name": "Value"
        }
      },
      [$.text(valueText)]),
      $.box({
        className: "property",
        attr: {
          "data-name": "Time"
        }
      },
      [$.text(formatTime(sensor.time))]),
      $.create("a",
      {
        className: "id",
        props: {
          href: `#devices/${deviceID}/sensors/${sensor.id}`
        }
      },
      [$.text(`ID ${sensor.id}`)])
    ]);
    return inflated;
  };

  inflateActuator = function(deviceID, actuator) {
    var inflated, nameText, valueText;
    if (actuator.value === null) {
      valueText = "(none)";
    } else {
      valueText = JSON.stringify(actuator.value, null, 2);
    }
    inflated = $.box({
      className: "box"
    }, [
      $.create("img",
      {
        props: {
          src: "img/actuator.png"
        }
      }),
      $.create("h2",
      {},
      [
        $.create("a",
        {
          props: {
            href: `#devices/${deviceID}/actuators/${actuator.id}`
          }
        },
        [nameText = $.text(actuator.name)])
      ]),
      $.box({
        className: "floating"
      },
      [
        $.create("img",
        {
          props: {
            src: "img/edit.svg",
            title: "Edit Name"
          },
          on: {
            click: async function() {
              var name,
        resp,
        text;
              name = prompt(`Please enter a new name for "${actuator.name}":`,
        actuator.name);
              if (name && name !== actuator.name) {
                resp = (await fetch(`devices/${deviceID}/actuators/${actuator.id}/name`,
        {
                  method: "POST",
                  body: name
                }));
                if (!resp.ok) {
                  text = (await resp.text());
                  alert("Error:\n" + text);
                  return;
                }
                nameText.textContent = name;
              }
            }
          }
        }),
        $.create("img",
        {
          props: {
            src: "img/delete.svg",
            title: "Delete"
          },
          on: {
            click: async function() {
              var resp,
        text;
              if (confirm(`Delete "${actuator.name}"?\nThis will also delete all actuator data points.`)) {
                resp = (await fetch(`devices/${deviceID}/actuators/${actuator.id}`,
        {
                  method: "DELETE"
                }));
                if (!resp.ok) {
                  text = (await resp.text());
                  alert("Error:\n" + text);
                  return;
                }
                $(inflated).remove();
              }
            }
          }
        })
      ]),
      $.box({
        className: "property",
        attr: {
          "data-name": "Value"
        }
      },
      [$.text(valueText)]),
      $.box({
        className: "property",
        attr: {
          "data-name": "Time"
        }
      },
      [$.text(formatTime(actuator.time))]),
      $.create("a",
      {
        className: "id",
        props: {
          href: `#devices/${deviceID}/actuators/${actuator.id}`
        }
      },
      [$.text(`ID ${actuator.id}`)])
    ]);
    return inflated;
  };

  //###################
  showSensor = async function(deviceID, sensorID) {
    var resp, resp2, sensor, tbody, value, values, virgin;
    resp = (await fetch(`devices/${deviceID}/sensors/${sensorID}`));
    if (!resp.ok) {
      showRespError(resp);
      return;
    }
    sensor = (await resp.json());
    heading.text(sensor.name);
    resp2 = (await fetch(`devices/${deviceID}/sensors/${sensorID}/values`));
    if (!resp2.ok) {
      showRespError(resp2);
      return;
    }
    values = (await resp2.json());
    subheading1.show().text(`${values.length} Values`);
    virgin = $.create("span", {
      className: "virgin",
      on: {
        click: async function() {
          var dpoint, err, resp3, text, time, value;
          value = prompt("Enter a new value (JSON):", "");
          if (!value) {
            return;
          }
          try {
            JSON.parse(value);
          } catch (error) {
            err = error;
            if (!confirm(`Value is not valid JSON! Send as string (with quotation marks)?\nValue: "${value}"`)) {
              return;
            }
            value = JSON.stringify(value);
          }
          resp3 = (await fetch(`devices/${deviceID}/sensors/${sensorID}/value`, {
            method: "POST",
            body: value
          }));
          if (!resp3.ok) {
            text = (await resp.text());
            alert("Error:\n" + text);
            return;
          }
          time = new Date();
          dpoint = $.create("tr", {}, [$.create("td", {}, [$.text(value)]), $.create("td", {}, [$.text(formatTime(time))])]);
          tbody.prepend(dpoint);
        }
      }
    }, [$.text("Push Value")]);
    subheading1.append(virgin);
    content1.text("");
    content1.append($.create("table", {}, [$.create("thead", {}, [$.create("tr", {}, [$.create("td", {}, [$.text("Values")]), $.create("td", {}, [$.text("Time")])])]), tbody = $.create("tbody", {}, [])]));
    for (value of values) {
      tbody.prepend($.create("tr", {}, [$.create("td", {}, [$.text(JSON.stringify(value.value, null, 2))]), $.create("td", {}, [$.text(formatTime(value.time))])]));
    }
    subheading2.hide();
    content2.hide();
  };

  showActuator = async function(deviceID, actuatorID) {
    var actuator, resp, resp2, tbody, value, values, virgin;
    resp = (await fetch(`devices/${deviceID}/actuators/${actuatorID}`));
    if (!resp.ok) {
      showRespError(resp);
      return;
    }
    actuator = (await resp.json());
    heading.text(actuator.name);
    resp2 = (await fetch(`devices/${deviceID}/actuators/${actuatorID}/values`));
    if (!resp2.ok) {
      showRespError(resp2);
      return;
    }
    values = (await resp2.json());
    subheading1.show().text(`${values.length} Values`);
    virgin = $.create("span", {
      className: "virgin",
      on: {
        click: async function() {
          var dpoint, err, resp3, text, time, value;
          value = prompt("Enter a new value (JSON):", "");
          if (!value) {
            return;
          }
          try {
            JSON.parse(value);
          } catch (error) {
            err = error;
            if (!confirm(`Value is not valid JSON! Send as string (with quotation marks)?\nValue: "${value}"`)) {
              return;
            }
            value = JSON.stringify(value);
          }
          resp3 = (await fetch(`devices/${deviceID}/actuators/${actuatorID}/value`, {
            method: "POST",
            body: value
          }));
          if (!resp3.ok) {
            text = (await resp.text());
            alert("Error:\n" + text);
            return;
          }
          time = new Date();
          dpoint = $.create("tr", {}, [$.create("td", {}, [$.text(value)]), $.create("td", {}, [$.text(formatTime(time))])]);
          tbody.prepend(dpoint);
        }
      }
    }, [$.text("Push Value")]);
    subheading1.append(virgin);
    content1.text("");
    content1.append($.create("table", {}, [$.create("thead", {}, [$.create("tr", {}, [$.create("td", {}, [$.text("Values")]), $.create("td", {}, [$.text("Time")])])]), tbody = $.create("tbody", {}, [])]));
    for (value of values) {
      tbody.prepend($.create("tr", {}, [$.create("td", {}, [$.text(JSON.stringify(value.value, null, 2))]), $.create("td", {}, [$.text(formatTime(value.time))])]));
    }
    subheading2.hide();
    content2.hide();
  };

  //###################
  showClouds = async function() {
    var cloud, clouds, id, resp;
    resp = (await fetch("clouds"));
    if (!resp.ok) {
      showRespError(resp);
      return;
    }
    clouds = (await resp.json());
    heading.text(`${(Object.keys(clouds).length)} Clouds`);
    subheading1.hide();
    subheading2.hide();
    content1.show().text("");
    content2.hide();
    for (id in clouds) {
      cloud = clouds[id];
      content1.append(inflateCloud(cloud));
    }
  };

  inflateCloud = function(cloud) {
    var checkPaused, inflated, inputMQTT, inputREST, inputToken, inputUsername, nameText;
    inflated = $.box({
      className: "box"
    }, [
      $.create("img",
      {
        props: {
          src: "img/device.png"
        }
      }),
      $.create("h2",
      {},
      [
        $.create("a",
        {
          props: {
            href: `#clouds/${cloud.id}`
          }
        },
        [nameText = $.text(cloud.id)])
      ]),
      $.box({
        className: "floating"
      },
      [
        $.create("img",
        {
          props: {
            src: "img/delete.svg",
            title: "Delete"
          },
          on: {
            click: async function() {
              var resp,
        text;
              if (confirm(`Delete "${cloud.id}"?`)) {
                resp = (await fetch(`clouds/${cloud.id}`,
        {
                  method: "DELETE"
                }));
                if (!resp.ok) {
                  text = (await resp.text());
                  alert("Error:\n" + text);
                  return;
                }
                $(inflated).remove();
              }
            }
          }
        })
      ]),
      $.box({
        className: "attr"
      },
      [
        $.create("label",
        {
          className: "label"
        },
        [$.text("Sync. Paused:")]),
        checkPaused = $.create("input",
        {
          attr: {
            type: "checkbox"
          },
          props: {
            checked: cloud.paused
          }
        })
      ]),
      $.box({
        className: "attr"
      },
      [
        $.create("label",
        {
          className: "label"
        },
        [$.text("Username:")]),
        inputUsername = $.create("input",
        {
          attr: {
            type: "text"
          },
          props: {
            value: cloud.username
          }
        })
      ]),
      $.box({
        className: "attr"
      },
      [
        $.create("label",
        {
          className: "label"
        },
        [$.text("Password:")]),
        inputToken = $.create("input",
        {
          attr: {
            type: "text"
          },
          props: {
            value: cloud.token
          }
        })
      ]),
      $.box({
        className: "attr"
      },
      [
        $.create("label",
        {
          className: "label"
        },
        [$.text("REST:")]),
        inputREST = $.create("input",
        {
          attr: {
            type: "text"
          },
          props: {
            value: cloud.rest
          }
        })
      ]),
      $.box({
        className: "attr"
      },
      [
        $.create("label",
        {
          className: "label"
        },
        [$.text("MQTT:")]),
        inputMQTT = $.create("input",
        {
          attr: {
            type: "text"
          },
          props: {
            value: cloud.mqtt
          }
        })
      ]),
      $.box({
        className: "attr"
      },
      [
        $.create("button",
        {
          className: "button",
          on: {
            click: async() => {
              var paused,
        resp,
        text;
              paused = checkPaused.checked;
              resp = (await fetch(`/clouds/${cloud.id}/paused`,
        {
                method: "POST",
                body: JSON.stringify(paused)
              }));
              if (resp.ok) {
                alert("OK");
              } else {
                text = (await resp.text());
                alert("Can not save:\n" + text);
              }
            }
          }
        },
        [$.text("Save Paused")]),
        $.create("button",
        {
          className: "button",
          on: {
            click: async() => {
              var resp,
        text;
              resp = (await fetch(`/clouds/${cloud.id}/username`,
        {
                method: "POST",
                body: JSON.stringify(inputUsername.value)
              }));
              if (!resp.ok) {
                text = (await resp.text());
                alert("Can not save:\n" + text);
                return;
              }
              resp = (await fetch(`/clouds/${cloud.id}/token`,
        {
                method: "POST",
                body: JSON.stringify(inputToken.value)
              }));
              if (resp.ok) {
                alert("OK");
              } else {
                text = (await resp.text());
                alert("Can not save:\n" + text);
              }
            }
          }
        },
        [$.text("Save Cred.")]),
        $.create("button",
        {
          className: "button",
          on: {
            click: async() => {
              var resp,
        text;
              resp = (await fetch(`/clouds/${cloud.id}/rest`,
        {
                method: "POST",
                body: JSON.stringify(inputREST.value)
              }));
              if (!resp.ok) {
                text = (await resp.text());
                alert("Can not save:\n" + text);
                return;
              }
              resp = (await fetch(`/clouds/${cloud.id}/mqtt`,
        {
                method: "POST",
                body: JSON.stringify(inputMQTT.value)
              }));
              if (resp.ok) {
                alert("OK");
              } else {
                text = (await resp.text());
                alert("Can not save:\n" + text);
              }
            }
          }
        },
        [$.text("Save Addr.")])
      ]),
      $.create("a",
      {
        className: "id",
        props: {
          href: `#clouds/${cloud.id}`
        }
      },
      [$.text(`ID ${cloud.id}`)])
    ]);
    return inflated;
  };

  //###################
  breadcrumbs = $("#breadcrumbs");

  showBreadcrumbs = function(hash) {
    var i, j;
    breadcrumbs.text("");
    j = 0;
    while (true) {
      i = hash.indexOf("/", j);
      if (i === -1) {
        breadcrumbs.append($.create("a", {
          props: {
            href: "#" + hash
          }
        }, [$.text(hash.slice(j))]));
        return;
      }
      breadcrumbs.append($.create("a", {
        props: {
          href: "#" + hash.slice(0, i)
        }
      }, [$.text(hash.slice(j, i))]));
      breadcrumbs.append($.text(" / "));
      j = i + 1;
    }
  };

  //###############################################################################
  formatTime = function(time) {
    var date, diff, now;
    date = new Date(time);
    now = new Date;
    diff = (now - date) / 1000;
    if (diff < 10) {
      return `just now, ${date.toLocaleString()}`;
    }
    if (diff < 60) {
      return `${Math.round(diff)} sec ago, ${date.toLocaleString()}`;
    }
    if (diff < 60 * 60) {
      return `${Math.round(diff / 60)} min ago, ${date.toLocaleString()}`;
    }
    if (diff < 60 * 60 * 24) {
      return `${Math.round(diff / 60 / 60)} hours ago, ${date.toLocaleString()}`;
    }
    return `${Math.round(diff / 60 / 60 / 24)} days ago, ${date.toLocaleString()}`;
    return "now";
  };

  window.addEventListener("popstate", function() {
    navigate(location.hash.slice(1));
  });

  navigate(location.hash.slice(1));

  //###############################################################################
  log = $("#logtext");

  tagRegexp = /^\[(\w+)\s*\]/;

  inflateLog = function(text) {
    var line, tag;
    line = $.box({
      className: "tag"
    }, [$.text(text)]);
    tag = tagRegexp.exec(text);
    if (tag !== null) {
      line.classList.add(`tag-${tag[1].toLowerCase()}`);
    }
    log.prepend(line);
  };

  logOpener = $("#log .opener");

  logOpener.on("click", () => {
    if ($.body.hasClass("log-open")) {
      logOpener.text("Show Log");
    } else {
      logOpener.text("Hide Log");
    }
    $.body.toggleClass("log-open");
  });

  //###############################################################################
  MQTT = Messaging;

  conncetMQTT = function() {
    var client, rnd;
    rnd = `${Math.random() * 1e6}`.slice(0, 6);
    client = new MQTT.Client(location.hostname, parseInt(location.port) || 80, "dashboard-" + rnd);
    client.onConnectionLost = function(resp) {
      if (resp.errorCode !== 0) {
        setStatus(false, `Connection lost: ${resp.errorMessage}`);
      } else {
        setStatus(false, "Connection lost.");
      }
      setTimeout(conncetMQTT, 2000);
    };
    client.onMessageArrived = function(msg) {
      if (msg.destinationName === "sys/log") {
        inflateLog(msg.payloadString);
      } else {
        console.log("onMessageArrived:", msg);
      }
    };
    // client.disconnect(); 
    client.connect({
      onSuccess: function() {
        setStatus(true, "Connected to Gateway.");
        client.subscribe("sys/log");
        refreshUptime();
      },
      // message = new MQTT.Message "Hello"
      // message.destinationName = "/World"
      // client.send message
      onFailure: function(err) {
        setStatus(false, `Can not connect to Gateway: ${err.errorMessage}`);
        setTimeout(conncetMQTT, 2000);
      }
    });
  };

  conncetMQTT();

  //###############################################################################
  status = $("#status");

  statusbar = $("#statusbar");

  setStatus = function(isOk, text) {
    status.text(text);
    statusbar.removeClass("ok err");
    if (isOk === true) {
      statusbar.addClass("ok");
    }
    if (isOk === false) {
      statusbar.addClass("err");
    }
  };

  timeSuffixes = {
    ms: 1,
    s: 1000,
    m: 60000,
    h: 3600000
  };

  uptimeText = $("#uptime");

  uptimeTicker = null;

  refreshUptime = async function() {
    var now, resp, setUptime, text, uptime;
    resp = (await fetch("sys/uptime"));
    if (!resp.ok) {
      return;
    }
    text = (await resp.text());
    now = new Date();
    if (!text.endsWith("ms")) {
      text.replace(/\d+(\.\d+)?\D+/g, function(seg) {
        var f;
        f = parseFloat(seg);
        return now -= f * timeSuffixes[seg[seg.length - 1]];
      });
    }
    uptime = new Date(now);
    setUptime = function() {
      text = `Gateway Start: ${formatTime(uptime)}`;
      if (uptimeText.text() !== text) {
        uptimeText.text(text);
      }
    };
    clearInterval(uptimeTicker);
    uptimeTicker = setInterval(setUptime, 1000);
    setUptime();
  };

  refreshUptime();

}).call(this);


//# sourceMappingURL=main.js.map
//# sourceURL=coffeescript