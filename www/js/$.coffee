class Query
  constructor: (elm) ->
    @$ = switch
      when ! elm then null
      when elm instanceof HTMLElement then elm
      when elm instanceof Text then elm
      when typeof elm is "string" then document.querySelector elm
      when elm instanceof Query then elm.$
      else elm
    return

  attr: (attrs, val) ->
    if arguments.length == 2 then return @attr {[attrs]: val}
    if typeof attrs == "string" then @$?.getAttribute attrs
    else
      for key, value of attrs
        @$.setAttribute key, value
    return this

  attrNS: (ns, attrs, val) ->
    if arguments.length == 3 then return @attrNS {[attrs]: val}, ns
    if typeof attrs == "string" then @$?.getAttributeNS ns, attrs
    else
      for key, value of attrs
        @$.setAttributeNS ns, key, value
    return this

  prop: (prop, val) ->
    if arguments.length == 2 then return @attr {[prop]: val}
    if typeof prop == "string" then @$?[prop]
    else $.assign @$, prop
    return this

  index: () ->
    i = 0
    $ = @$
    i++ while ($ = $.previousSibling) != null
    return i

  append: (elm...) ->
    for child in elm
      if child != null
        @$.appendChild child
    return this

  appendTo: (target) ->
    $(target).$?.appendChild @$
    return this

  remove: () ->
    if @$ != null
      @$.parentNode?.removeChild @$
    return

  on: (event, handler, options={}) ->
    if Array.isArray handler
      @$.addEventListener event, h, options for h in handler
    else
      @$.addEventListener event, handler, options
    return this

  once: (event, handler, options={}) ->
    options.once = true
    @on event, handler, options
    return this

  off: (event, handler, options) ->
    @$.removeEventListener event, handler, options
    return this

  emit: (event, data={}) ->
    evt = new CustomEvent event, data
    cancelled = !@$.dispatchEvent evt
    target = evt.path[evt.path.length-1]
    if data.ignoreHost and ! cancelled and target.nodeType == 11 # DOCUMENT_FRAGMENT_NODE
      $ target.host
        .emit event, data
    return this

  addClass: (cls) ->
    @$.classList.add ... cls.split " "

  removeClass: (cls) ->
    @$.classList.remove ... cls.split " "

  hasClass: (cls) ->
    for c in cls.split " "
      if @$.classList.contains c
          return true
    return false

  find: (sel) -> (@$.querySelector sel) || null

  findAll: (sel) -> @$.querySelectorAll sel

  text: (text) ->
    @$.textContent = text
    return text

  show: () ->
    @$.style.display = ""
    return this

  hide: () ->
    @$.style.display = "none"
    return this

  style: (style, val) ->
    if arguments.length == 2
      @$.style[style] = val
      return this
    if typeof style == "string"
      return @$.style[prop]
    else
      $.assign @$.style, style
      return this

$ = (elm) -> new Query elm

$.html = $ document.body.parentElement
$.window = $ window
$.document = $ document
$.body = $ document.body
$.head = $ document.head

################################################################################

$.drag = (event, d, onDrag, onEnd) ->
  sx = event.screenX
  sy = event.screenY
  dragging = false
  onMouseMove = (event) =>
    dx = event.screenX-sx
    dy = event.screenY-sy
    if !dragging
      if Math.abs(dx)+Math.abs(dy) < d
        return
      sx = event.screenX
      sy = event.screenY
    onDrag event, dx, dy, dragging
    dragging = true
    return
  options = {capture: true}
  $.window.on "mousemove", onMouseMove, options
  $.window.once "mouseup", (event) =>
    $.window.off "mousemove", onMouseMove, options
    dx = event.screenX-sx
    dy = event.screenY-sy
    onEnd event, dx, dy, dragging
    return
  return

$.assign = (a, b) ->
  return b if a == undefined || a == null
  return a if b == undefined || b == null
  Object.keys(b).forEach (key) =>
    if typeof b[key] == "object"
      $.assign a[key], b[key]
    else
      a[key] = b[key]
  return a

$.extend = (a, b) ->
  return b if a == undefined || a == null
  return a if b == undefined || b == null
  Object.keys(b).forEach (key) =>
    if typeof b[key] == "object"
      $.extend a[key], b[key]
    else
      a[key] = a[key]+" "+b[key]
  return a

$.text = (text) -> document.createTextNode(text)
$.$text = (text) -> $ document.createTextNode(text)

$.createNS = (ns, tag, attr, children) -> $.$createNS(ns, tag, attr, children).$
$.$createNS = (ns, tag, attr, children) ->
  elm = $ document.createElementNS ns, tag
  elm.attr attr if attr
  # elm.prop attr if attr
  elm.append ...children if children
  return elm

$.create = (tag, props, children) -> $.$create(tag, props, children).$
$.$create = (tag, props, children) ->
  elm = $ document.createElement tag
  if props
    if "src" of props
      console.warn "props.src should be moved to props.attr.src"
    if "className" of props
      elm.$.className = props.className
    if "state" of props
      elm.val props.state
    if "props" of props
      elm.prop props.props
    if "attr" of props
      elm.attr props.attr
    if "style" of props
      elm.prop {style: props.style}
    if "on" of props
      for event, listeners of props.on
        elm.on event, listeners
  if children
    elm.append children...
  return elm

$.box = (props, children) -> $.$box(props, children).$
$.$box = (props, children) -> $.$create "div", props, children

################################################################################

$.requireScript = (src) ->

  console.warn "$.requireStyle is deprecated."

  if (src.startsWith "./") || (src.startsWith "../")
    pkg = document.currentScript.getAttribute "data-pkg"
    src = "/fs/lang/"+pkg+"/"+src
  else
    if (src.startsWith "http://") || (src.startsWith "https://") || (src.startsWith "/")
      # src = src
    else
      src = "/fs/lang/"+src

  return new Promise (resolve, reject) =>
    $.$create "script",
        attr: src: src
      .on "error", (event) =>
        reject event
        return
      .on "load", (event) =>
        resolve event
        return
      .appendTo $.head

$.requireStyle = (src) ->

  console.warn "$.requireStyle is deprecated. Use require.Style(..)\nSource: "+src

  if (src.startsWith "./") || (src.startsWith "../")
    pkg = document.currentScript.getAttribute "data-pkg"
    src = "/fs/lang/"+pkg+"/"+src
  else
    if (src.startsWith "http://") || (src.startsWith "https://") || (src.startsWith "/")
      # src = src
    else
      src = "fs/lang/"+src

  return new Promise (resolve, reject) =>
    $.$create "link",
        attr:
          rel: "stylesheet"
          href: src
      .on "error", (event) =>
        reject event
        return
      .on "load", () =>
        resolve event
        return
      .appendTo $.head

################################################################################

(() ->
  counter = 0
  style = $.create "style"
  style.title  ="autogen styles"
  $.head.append style
  doc = style.sheet
  Object.defineProperty doc, "title",
    value: "autogen styles"

  $.style = (rule) =>
    cls = counter.toString 36
    r = Object.keys(rule)
      .map (key) =>
        if key[0] == ":"
          pseudo = rule[key]
          rr = Object.keys(pseudo)
            .map (key) => "#{key}: #{pseudo[key]}"
            .join ";\n  "
          doc.insertRule "._#{cls}#{key} {\n  #{rr};\n}"
          return ""
        return "#{key}: #{rule[key]}"
      .join ";\n  "
    doc.insertRule "._#{cls} {\n  #{r};\n}"
    counter++
    return "_"+cls

  $.rawStyle = (rule) =>
    doc.insertRule rule

  $.makeStyle = (states) =>
    cls = counter.toString 36
    for state, styles of states
      rule = Object.keys styles
        .map (key) => "#{key}: #{styles[key]}"
        .join ";\n  "
      if state == "default"
        doc.insertRule "._#{cls} {  #{rule};\n}"
      else
        doc.insertRule "._#{cls}#{state} {  #{rule};\n}"
    counter++
    return "_"+cls

)()


################################################################################

$.platform = () -> switch
  when navigator.platform.startsWith "Win" then "windows"
  when navigator.platform.startsWith "Mac" then "mac"
  when navigator.platform.includes "Linux" then "linux"
  else ""
