switch $.platform()
  when "windows"
    $.body.addClass "windows"
  when "linux"
    $.body.addClass "linux"
  when "mac"
    $.body.addClass "mac"

ontologies = null

container = $ "#ontologies"

main = () ->
  file = await fetch "ontologies.json"
  ontologies = await file.json()
  inflate ontologies
  return

svgNS = "http://www.w3.org/2000/svg"
xlinkNS = "http://www.w3.org/1999/xlink"

inflate = (ontologies) ->
  for id, device of ontologies.sensingDevices
    inflateSensingDevice id, device
  return

inflateSensingDevice = (id, device) ->
  svgRef = $.createNS svgNS, "use", {}
  svgRef.setAttributeNS xlinkNS, "xlink:href", "ontologies.svg#"+device.icon
  svgIcon = $.createNS svgNS, "svg",
    class: "icon"
  , [svgRef]
  box = $.box
    className: "sensing-device"
  , [
    svgIcon
    $.create "h2", {}, [$.text device.label]
    $.create "h3", {}, [$.text "Quantities"]
    $.create "ul", {},
      device.quantities.map (quantity) ->
        $.create "li", {}, [
          $.text ontologies.quantities[quantity].label
          ... ontologies.quantities[quantity].units.map (unit) ->
            $.create "span",
              className: "unit code"
            , [$.text ontologies.units[unit].label]
        ]
  ]
  container.append box
  return

main()