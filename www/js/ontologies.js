(function() {
  var container, inflate, inflateSensingDevice, main, ontologies, svgNS, xlinkNS;

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

  ontologies = null;

  container = $("#ontologies");

  main = async function() {
    var file;
    file = (await fetch("ontologies.json"));
    ontologies = (await file.json());
    inflate(ontologies);
  };

  svgNS = "http://www.w3.org/2000/svg";

  xlinkNS = "http://www.w3.org/1999/xlink";

  inflate = function(ontologies) {
    var device, id, ref;
    ref = ontologies.sensingDevices;
    for (id in ref) {
      device = ref[id];
      inflateSensingDevice(id, device);
    }
  };

  inflateSensingDevice = function(id, device) {
    var box, svgIcon, svgRef;
    svgRef = $.createNS(svgNS, "use", {});
    svgRef.setAttributeNS(xlinkNS, "xlink:href", "ontologies.svg#" + device.icon);
    svgIcon = $.createNS(svgNS, "svg", {
      class: "icon"
    }, [svgRef]);
    box = $.box({
      className: "sensing-device"
    }, [
      svgIcon,
      $.create("h2",
      {},
      [$.text(device.label)]),
      $.create("h3",
      {},
      [$.text("Quantities")]),
      $.create("ul",
      {},
      device.quantities.map(function(quantity) {
        return $.create("li",
      {},
      [
          $.text(ontologies.quantities[quantity].label),
          ...ontologies.quantities[quantity].units.map(function(unit) {
            return $.create("span",
          {
              className: "unit code"
            },
          [$.text(ontologies.units[unit].label)]);
          })
        ]);
      }))
    ]);
    container.append(box);
  };

  main();

}).call(this);


//# sourceMappingURL=ontologies.js.map
//# sourceURL=coffeescript