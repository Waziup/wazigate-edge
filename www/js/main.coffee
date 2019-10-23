switch $.platform()
    when "windows"
        $.body.addClass "windows"
    when "linux"
        $.body.addClass "linux"
    when "mac"
        $.body.addClass "mac"

navigate = (hash) ->
    if hash == "" 
        location.hash = "#devices"
        return

    if location.hash != "#"+hash
        location.hash = "#"+hash
        return

    showBreadcrumbs hash

    if hash == "devices"
        showDevices()
        return

    if hash == "clouds"
        showClouds()
        return
    
    match = hash.match /^devices\/(\w+)$/
    if match != null
        showDevice match[1]
        return

    match = hash.match /^devices\/(\w+)\/sensors$/
    if match != null
        location.hash = "#devices/#{match[1]}"
        return

    match = hash.match /^devices\/(\w+)\/actuators$/
    if match != null
        location.hash = "#devices/#{match[1]}"
        return

    match = hash.match /^devices\/(\w+)\/sensors\/(\w+)$/
    if match != null
        showSensor match[1], match[2]
        return

    match = hash.match /^devices\/(\w+)\/actuators\/(\w+)$/
    if match != null
        showActuator match[1], match[2]
        return

    # location.hash = "#devices"
    return

################################################################################

heading = $ "#heading"
content1 = $ "#content1"
subheading1 = $ "#subheading1"
content2 = $ "#content2"
subheading2 = $ "#subheading2"

showDevices = () ->
    resp = await fetch "devices"
    if ! resp.ok
        showRespError resp
        return
    devices = await resp.json()

    heading.text "#{devices.length} Devices"

    subheading1.hide()
    subheading2.hide()
    content1.show().text ""
    content2.hide()

    for device from devices
        content1.append inflateDevice device

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
    return

inflateDevice = (device) ->
    inflated = $.box
        className: "box"
    , [
        $.create "img",
            props: src: "img/device.png"
        $.create "h2", {}
        , [
            $.create "a",
                props: href: "#devices/#{device.id}"
            , [ nameText = $.text device.name ]
        ]
        $.box
            className: "floating",
        , [
            $.create "img",
                props:
                    src: "img/edit.svg"
                    title: "Edit Name"
                on: click: () -> 
                    name = prompt "Please enter a new name for \"#{device.name}\":", device.name
                    if name && name != device.name
                        resp = await fetch "devices/#{device.id}/name",
                            method: "POST"
                            body: name
                        if ! resp.ok
                            text = await resp.text()
                            alert "Error:\n"+text
                            return
                        nameText.textContent = name
                    return
            $.create "img",
                props:
                    src: "img/delete.svg"
                    title: "Delete"
                on: click: () ->
                    if confirm "Delete \"#{device.name}\"?\nThis will also delete all device data points."
                        resp = await fetch "devices/#{device.id}",
                            method: "DELETE"
                        if ! resp.ok
                            text = await resp.text()
                            alert "Error:\n"+text
                            return
                        $(inflated).remove()
                    return
        ]
        $.create "a",
            className: "id"
            props: href: "#devices/#{device.id}"
        , [ $.text "ID #{device.id}" ]
    ]
    return inflated

####################

showDevice = (deviceID) ->
    resp = await fetch "devices/#{deviceID}"
    if ! resp.ok
        showRespError resp
        return
    device = await resp.json()

    heading.text device.name
    
    content1.show().text ""
    subheading1.show().text "#{device.sensors.length} Sensors"

    for sensor from device.sensors
        content1.append inflateSensor deviceID, sensor

     virgin1 =  $.box
        className: "virgin"
        on: click: () ->
            name = prompt "Please enter sensor name:", "New Sensor"
            return if ! name
            sensor = 
                name: name
                value: null
                time: new Date()
            resp = await fetch "devices/#{deviceID}/sensors",
                method: "POST"
                body: JSON.stringify sensor
            if ! resp.ok
                text = await resp.text()
                alert "Error:\n"+text
                return
            id = await resp.text()
            sensor.id = id
            inflated = inflateSensor device.id, sensor
            content1.$.insertBefore inflated, virgin1
            return        
    , [$.text "Create new Sensor"]

    content1.append virgin1

    ####

    content2.show().text ""
    subheading2.show().text "#{device.actuators.length} Actuators"

    for actuator from device.actuators
        content2.append inflateActuator deviceID, actuator

     virgin2 =  $.box
        className: "virgin"
        on: click: () ->
            name = prompt "Please enter actuator name:", "New Actuator"
            return if ! name
            actuator = 
                name: name
                value: null
                time: new Date()
            resp = await fetch "devices/#{deviceID}/actuators",
                method: "POST"
                body: JSON.stringify actuator
            if ! resp.ok
                text = await resp.text()
                alert "Error:\n"+text
                return
            id = await resp.text()
            actuator.id = id
            inflated = inflateActuator device.id, actuator
            content2.$.insertBefore inflated, virgin2
            return        
    , [$.text "Create new Actuator"]

    content2.append virgin2
    return

inflateSensor = (deviceID, sensor) ->
    if sensor.value == null
        valueText = "(none)"
    else
        valueText = JSON.stringify sensor.value, null, 2

    inflated = $.box
        className: "box"
    , [
        $.create "img",
            props: src: "img/sensor.png"
        $.create "h2", {}
        , [
            $.create "a",
                props: href: "#devices/#{deviceID}/sensors/#{sensor.id}"
            , [ nameText = $.text sensor.name ]
        ]
        $.box
            className: "floating",
        , [
            $.create "img",
                props:
                    src: "img/edit.svg"
                    title: "Edit Name"
                on: click: () -> 
                    name = prompt "Please enter a new name for \"#{sensor.name}\":", sensor.name
                    if name && name != sensor.name
                        resp = await fetch "devices/#{deviceID}/sensors/#{sensor.id}/name",
                            method: "POST"
                            body: name
                        if ! resp.ok
                            text = await resp.text()
                            alert "Error:\n"+text
                            return
                        nameText.textContent = name
                    return
            $.create "img",
                props:
                    src: "img/delete.svg"
                    title: "Delete"
                on: click: () ->
                    if confirm "Delete \"#{sensor.name}\"?\nThis will also delete all sensor data points."
                        resp = await fetch "devices/#{deviceID}/sensors/#{sensor.id}",
                            method: "DELETE"
                        if ! resp.ok
                            text = await resp.text()
                            alert "Error:\n"+text
                            return
                        $(inflated).remove()
                    return
        ]
        $.box
            className: "property"
            attr: "data-name": "Value"
        , [ $.text valueText ]

        $.box
            className: "property"
            attr: "data-name": "Time"
        , [ $.text formatTime sensor.time ]

        $.create "a",
            className: "id"
            props: href: "#devices/#{deviceID}/sensors/#{sensor.id}"
        , [ $.text "ID #{sensor.id}" ]
    ]
    return inflated

inflateActuator = (deviceID, actuator) ->
    if actuator.value == null
        valueText = "(none)"
    else
        valueText = JSON.stringify actuator.value, null, 2

    inflated = $.box
        className: "box"
    , [
        $.create "img",
            props: src: "img/actuator.png"
        $.create "h2", {}
        , [
            $.create "a",
                props: href: "#devices/#{deviceID}/actuators/#{actuator.id}"
            , [ nameText = $.text actuator.name ]
        ]
        $.box
            className: "floating",
        , [
            $.create "img",
                props:
                    src: "img/edit.svg"
                    title: "Edit Name"
                on: click: () -> 
                    name = prompt "Please enter a new name for \"#{actuator.name}\":", actuator.name
                    if name && name != actuator.name
                        resp = await fetch "devices/#{deviceID}/actuators/#{actuator.id}/name",
                            method: "POST"
                            body: name
                        if ! resp.ok
                            text = await resp.text()
                            alert "Error:\n"+text
                            return
                        nameText.textContent = name
                    return
            $.create "img",
                props:
                    src: "img/delete.svg"
                    title: "Delete"
                on: click: () ->
                    if confirm "Delete \"#{actuator.name}\"?\nThis will also delete all actuator data points."
                        resp = await fetch "devices/#{deviceID}/actuators/#{actuator.id}",
                            method: "DELETE"
                        if ! resp.ok
                            text = await resp.text()
                            alert "Error:\n"+text
                            return
                        $(inflated).remove()
                    return
        ]
        $.box
            className: "property"
            attr: "data-name": "Value"
        , [ $.text valueText ]

        $.box
            className: "property"
            attr: "data-name": "Time"
        , [ $.text formatTime actuator.time ]

        $.create "a",
            className: "id"
            props: href: "#devices/#{deviceID}/actuators/#{actuator.id}"
        , [ $.text "ID #{actuator.id}" ]
    ]
    return inflated

####################

showSensor = (deviceID, sensorID) ->
    resp = await fetch "devices/#{deviceID}/sensors/#{sensorID}"
    if ! resp.ok
        showRespError resp
        return
    sensor = await resp.json()

    heading.text sensor.name

    resp2 = await fetch "devices/#{deviceID}/sensors/#{sensorID}/values"
    if ! resp2.ok
        showRespError resp2
        return

    values = await resp2.json()
    subheading1.show().text "#{values.length} Values"

    virgin = $.create "span",
        className: "virgin"
        on: click: () ->
            value = prompt "Enter a new value (JSON):", ""
            return if ! value
            try
                JSON.parse value
            catch err
                if ! confirm "Value is not valid JSON! Send as string (with quotation marks)?\nValue: \"#{value}\""
                    return
                value = JSON.stringify value
            resp3 = await fetch "devices/#{deviceID}/sensors/#{sensorID}/value",
                method: "POST"
                body: value
            if ! resp3.ok
                text = await resp.text()
                alert "Error:\n"+text
                return
            time = new Date()
            dpoint = $.create "tr", {}, [
                $.create "td", {}, [$.text value]
                $.create "td", {}, [$.text formatTime time]
            ]
            tbody.prepend dpoint
            return

    , [ $.text "Push Value"]
    subheading1.append virgin

    content1.text ""
    content1.append $.create "table", {}, [
        $.create "thead", {}, [
            $.create "tr", {}, [
                $.create "td", {}, [$.text "Values"]
                $.create "td", {}, [$.text "Time"]
            ]
        ]
        tbody = $.create "tbody", {}, []
    ]

    for value from values
        tbody.prepend $.create "tr", {}, [
            $.create "td", {}, [$.text JSON.stringify value.value, null, 2]
            $.create "td", {}, [$.text formatTime value.time]
        ]

    subheading2.hide()
    content2.hide()
    return

showActuator = (deviceID, actuatorID) ->
    resp = await fetch "devices/#{deviceID}/actuators/#{actuatorID}"
    if ! resp.ok
        showRespError resp
        return
    actuator = await resp.json()

    heading.text actuator.name

    resp2 = await fetch "devices/#{deviceID}/actuators/#{actuatorID}/values"
    if ! resp2.ok
        showRespError resp2
        return

    values = await resp2.json()
    subheading1.show().text "#{values.length} Values"

    virgin = $.create "span",
        className: "virgin"
        on: click: () ->
            value = prompt "Enter a new value (JSON):", ""
            return if ! value
            try
                JSON.parse value
            catch err
                if ! confirm "Value is not valid JSON! Send as string (with quotation marks)?\nValue: \"#{value}\""
                    return
                value = JSON.stringify value
            resp3 = await fetch "devices/#{deviceID}/actuators/#{actuatorID}/value",
                method: "POST"
                body: value
            if ! resp3.ok
                text = await resp.text()
                alert "Error:\n"+text
                return
            time = new Date()
            dpoint = $.create "tr", {}, [
                $.create "td", {}, [$.text value]
                $.create "td", {}, [$.text formatTime time]
            ]
            tbody.prepend dpoint
            return

    , [ $.text "Push Value"]
    subheading1.append virgin

    content1.text ""
    content1.append $.create "table", {}, [
        $.create "thead", {}, [
            $.create "tr", {}, [
                $.create "td", {}, [$.text "Values"]
                $.create "td", {}, [$.text "Time"]
            ]
        ]
        tbody = $.create "tbody", {}, []
    ]

    for value from values
        tbody.prepend $.create "tr", {}, [
            $.create "td", {}, [$.text JSON.stringify value.value, null, 2]
            $.create "td", {}, [$.text formatTime value.time]
        ]

    subheading2.hide()
    content2.hide()
    return

####################

showClouds = () ->
    resp = await fetch "clouds"
    if ! resp.ok
        showRespError resp
        return
    clouds = await resp.json()

    heading.text "#{Object.keys(clouds).length} Clouds"

    subheading1.hide()
    subheading2.hide()
    content1.show().text ""
    content2.hide()

    for id, cloud of clouds
        content1.append inflateCloud cloud
    ###
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
    ###
    return

inflateCloud = (cloud) ->
    inflated = $.box
        className: "box"
    , [
        $.create "img",
            props: src: "img/device.png"
        $.create "h2", {}
        , [
            $.create "a",
                props: href: "#clouds/#{cloud.id}"
            , [ nameText = $.text cloud.id ]
        ]
        $.box
            className: "floating",
        , [
            $.create "img",
                props:
                    src: "img/delete.svg"
                    title: "Delete"
                on: click: () ->
                    if confirm "Delete \"#{cloud.id}\"?"
                        resp = await fetch "clouds/#{cloud.id}",
                            method: "DELETE"
                        if ! resp.ok
                            text = await resp.text()
                            alert "Error:\n"+text
                            return
                        $(inflated).remove()
                    return
        ]
        $.box
            className: "attr"
        , [
            $.create "label", 
                className: "label"
            , [$.text "Sync. Paused:"]
            checkPaused = $.create "input",
                attr: type: "checkbox"
                props: checked: cloud.paused
        ]
        $.box 
            className: "attr"
        , [
            $.create "label", 
                className: "label"
            , [$.text "Username:"]
            inputUsername = $.create "input",
                attr: type: "text"
                props: value: cloud.credentials.username
        ]
        $.box 
            className: "attr"
        , [
            $.create "label", 
                className: "label"
            , [$.text "Password:"]
            inputToken = $.create "input",
                attr: type: "text"
                props: value: cloud.credentials.token
        ]
        $.box 
            className: "attr"
        , [
            $.create "label", 
                className: "label"
            , [$.text "REST:"]
            inputREST = $.create "input",
                attr: type: "text"
                props: value: cloud.rest
        ]
        $.box 
            className: "attr"
        , [
            $.create "label", 
                className: "label"
            , [$.text "MQTT:"]
            inputMQTT = $.create "input",
                attr: type: "text"
                props: value: cloud.mqtt
        ]
        $.box 
            className: "attr"
        , [
            $.create "button", 
                className: "button"
                on: click: () =>
                    paused = checkPaused.checked
                    resp = await fetch "/clouds/#{cloud.id}/paused",
                        method: "POST"
                        body: JSON.stringify paused
                    if resp.ok
                        alert "OK"
                    else
                        text = await resp.text()
                        alert "Can not save:\n"+text
                    return
            , [$.text "Save Paused"]
            $.create "button", 
                className: "button"
                on: click: () =>
                    creds = 
                        username: inputUsername.value
                        token: inputToken.value
                    resp = await fetch "/clouds/#{cloud.id}/credentials",
                        method: "POST"
                        body: JSON.stringify creds
                    if resp.ok
                        alert "OK"
                    else
                        text = await resp.text()
                        alert "Can not save:\n"+text
                    return
            , [$.text "Save Cred."]
            $.create "button", 
                className: "button"
                on: click: () =>
                    resp = await fetch "/clouds/#{cloud.id}/rest",
                        method: "POST"
                        body: JSON.stringify inputREST.value
                    if ! resp.ok
                        text = await resp.text()
                        alert "Can not save:\n"+text
                        return
                    resp = await fetch "/clouds/#{cloud.id}/mqtt",
                        method: "POST"
                        body: JSON.stringify inputMQTT.value
                    if resp.ok
                        alert "OK"
                    else
                        text = await resp.text()
                        alert "Can not save:\n"+text
                    return
            , [$.text "Save Addr."]
        ]
        $.create "a",
            className: "id"
            props: href: "#clouds/#{cloud.id}"
        , [ $.text "ID #{cloud.id}" ]
    ]
    return inflated

####################

breadcrumbs = $ "#breadcrumbs"

showBreadcrumbs = (hash) ->
    breadcrumbs.text ""
    j = 0
    while true
        i = hash.indexOf "/", j
        if i == -1
            breadcrumbs.append $.create "a",
                props: href: "#"+hash
            , [
                $.text hash[j...]
            ]
            return
        breadcrumbs.append $.create "a",
                props: href: "#"+hash[...i]
        , [
            $.text hash[j...i]
        ]
        breadcrumbs.append $.text " / "
        j = i+1

################################################################################

formatTime = (time) ->
    date = new Date time
    now = new Date
    diff = (now-date)/1000

    if diff < 10 then return "just now, #{date.toLocaleString()}"
    if diff < 60 then return "#{Math.round diff} sec ago, #{date.toLocaleString()}"
    if diff < 60*60 then return "#{Math.round diff/60} min ago, #{date.toLocaleString()}"
    if diff < 60*60*24 then return "#{Math.round diff/60/60} hours ago, #{date.toLocaleString()}"
    return "#{Math.round diff/60/60/24} days ago, #{date.toLocaleString()}"

    return "now"

window.addEventListener "popstate", () ->
    navigate location.hash[1...]
    return

navigate location.hash[1...]

################################################################################

log = $ "#logtext"
tagRegexp = /^\[(\w+)\s*\]/

inflateLog = (text) ->

    line = $.box
        className: "tag"
    , [$.text text]

    tag = tagRegexp.exec text
    if tag != null
        line.classList.add "tag-#{tag[1].toLowerCase()}"

    log.prepend line
    return    


logOpener = $ "#log .opener"
logOpener.on "click", () =>
    if $.body.hasClass "log-open"
        logOpener.text "Show Log"
    else
        logOpener.text "Hide Log"
    $.body.toggleClass "log-open"
    return

################################################################################

MQTT = Messaging

conncetMQTT = () ->

    rnd = "#{Math.random()*1e6}"[...6]
    client = new MQTT.Client location.hostname, parseInt(location.port)||80, "dashboard-"+rnd

    client.onConnectionLost = (resp) ->
        if resp.errorCode != 0
            setStatus false, "Connection lost: #{resp.errorMessage}"
         else
            setStatus false, "Connection lost."
        setTimeout conncetMQTT, 2000
        return

    client.onMessageArrived = (msg) ->
        if msg.destinationName == "sys/log"
            inflateLog msg.payloadString
        else
            console.log "onMessageArrived:", msg
        return
        # client.disconnect(); 

    client.connect
        onSuccess: () ->
            setStatus true, "Connected to Gateway."
            client.subscribe "sys/log"
            refreshUptime()

            # message = new MQTT.Message "Hello"
            # message.destinationName = "/World"
            # client.send message
            return
        onFailure: (err) ->
            setStatus false, "Can not connect to Gateway: #{err.errorMessage}"
            setTimeout conncetMQTT, 2000
            return
    return

conncetMQTT()

################################################################################

status = $ "#status"
statusbar = $ "#statusbar"

setStatus = (isOk, text) ->
    status.text text
    statusbar.removeClass "ok err"
    if isOk == true
        statusbar.addClass "ok"
    if isOk == false
        statusbar.addClass "err"
    return


timeSuffixes = 
    ms: 1
    s: 1000
    m: 60000
    h: 3600000

uptimeText = $ "#uptime"

uptimeTicker = null

refreshUptime = () ->
    resp = await fetch "sys/uptime"
    return if ! resp.ok
    text = await resp.text()
    now = new Date()
    if ! text.endsWith "ms"
        text.replace /\d+(\.\d+)?\D+/g, (seg) ->
            f = parseFloat seg
            now -= f*timeSuffixes[seg[seg.length-1]]
    uptime = new Date now

    setUptime = () ->
        text = "Gateway Start: #{formatTime uptime}"
        if uptimeText.text() != text
            uptimeText.text text
        return

    clearInterval uptimeTicker
    uptimeTicker = setInterval setUptime, 1000
    setUptime()
    return


refreshUptime()