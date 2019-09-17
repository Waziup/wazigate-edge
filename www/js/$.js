(function() {
  var $, Query;

  Query = class Query {
    constructor(elm) {
      this.$ = (function() {
        switch (false) {
          case !!elm:
            return null;
          case !(elm instanceof HTMLElement):
            return elm;
          case !(elm instanceof Text):
            return elm;
          case typeof elm !== "string":
            return document.querySelector(elm);
          case !(elm instanceof Query):
            return elm.$;
          default:
            return elm;
        }
      })();
      return;
    }

    attr(attrs, val) {
      var key, ref, value;
      if (arguments.length === 2) {
        return this.attr({
          [attrs]: val
        });
      }
      if (typeof attrs === "string") {
        if ((ref = this.$) != null) {
          ref.getAttribute(attrs);
        }
      } else {
        for (key in attrs) {
          value = attrs[key];
          this.$.setAttribute(key, value);
        }
      }
      return this;
    }

    attrNS(ns, attrs, val) {
      var key, ref, value;
      if (arguments.length === 3) {
        return this.attrNS({
          [attrs]: val
        }, ns);
      }
      if (typeof attrs === "string") {
        if ((ref = this.$) != null) {
          ref.getAttributeNS(ns, attrs);
        }
      } else {
        for (key in attrs) {
          value = attrs[key];
          this.$.setAttributeNS(ns, key, value);
        }
      }
      return this;
    }

    prop(prop, val) {
      var ref;
      if (arguments.length === 2) {
        return this.attr({
          [prop]: val
        });
      }
      if (typeof prop === "string") {
        if ((ref = this.$) != null) {
          ref[prop];
        }
      } else {
        $.assign(this.$, prop);
      }
      return this;
    }

    index() {
      var $, i;
      i = 0;
      $ = this.$;
      while (($ = $.previousSibling) !== null) {
        i++;
      }
      return i;
    }

    append(...elm) {
      var child, j, len;
      for (j = 0, len = elm.length; j < len; j++) {
        child = elm[j];
        if (child !== null) {
          this.$.appendChild(child);
        }
      }
      return this;
    }

    prepend(...elm) {
      var child, j, len;
      for (j = 0, len = elm.length; j < len; j++) {
        child = elm[j];
        if (child !== null) {
          this.$.prepend(child);
        }
      }
      return this;
    }

    appendTo(target) {
      var ref;
      if ((ref = $(target).$) != null) {
        ref.appendChild(this.$);
      }
      return this;
    }

    remove() {
      var ref;
      if (this.$ !== null) {
        if ((ref = this.$.parentNode) != null) {
          ref.removeChild(this.$);
        }
      }
    }

    on(event, handler, options = {}) {
      var h, j, len;
      if (Array.isArray(handler)) {
        for (j = 0, len = handler.length; j < len; j++) {
          h = handler[j];
          this.$.addEventListener(event, h, options);
        }
      } else {
        this.$.addEventListener(event, handler, options);
      }
      return this;
    }

    once(event, handler, options = {}) {
      options.once = true;
      this.on(event, handler, options);
      return this;
    }

    off(event, handler, options) {
      this.$.removeEventListener(event, handler, options);
      return this;
    }

    emit(event, data = {}) {
      var cancelled, evt, target;
      evt = new CustomEvent(event, data);
      cancelled = !this.$.dispatchEvent(evt);
      target = evt.path[evt.path.length - 1];
      if (data.ignoreHost && !cancelled && target.nodeType === 11) { // DOCUMENT_FRAGMENT_NODE
        $(target.host).emit(event, data);
      }
      return this;
    }

    addClass(cls) {
      return this.$.classList.add(...cls.split(" "));
    }

    removeClass(cls) {
      return this.$.classList.remove(...cls.split(" "));
    }

    toggleClass(cls) {
      return this.$.classList.toggle(...cls.split(" "));
    }

    hasClass(cls) {
      var c, j, len, ref;
      ref = cls.split(" ");
      for (j = 0, len = ref.length; j < len; j++) {
        c = ref[j];
        if (this.$.classList.contains(c)) {
          return true;
        }
      }
      return false;
    }

    find(sel) {
      return (this.$.querySelector(sel)) || null;
    }

    findAll(sel) {
      return this.$.querySelectorAll(sel);
    }

    text(text) {
      if (arguments.length === 0) {
        return this.$.textContent;
      }
      this.$.textContent = text;
      return text;
    }

    show() {
      this.$.style.display = "";
      return this;
    }

    hide() {
      this.$.style.display = "none";
      return this;
    }

    style(style, val) {
      if (arguments.length === 2) {
        this.$.style[style] = val;
        return this;
      }
      if (typeof style === "string") {
        return this.$.style[prop];
      } else {
        $.assign(this.$.style, style);
        return this;
      }
    }

  };

  $ = function(elm) {
    return new Query(elm);
  };

  $.html = $(document.body.parentElement);

  $.window = $(window);

  $.document = $(document);

  $.body = $(document.body);

  $.head = $(document.head);

  //###############################################################################
  $.drag = function(event, d, onDrag, onEnd) {
    var dragging, onMouseMove, options, sx, sy;
    sx = event.screenX;
    sy = event.screenY;
    dragging = false;
    onMouseMove = (event) => {
      var dx, dy;
      dx = event.screenX - sx;
      dy = event.screenY - sy;
      if (!dragging) {
        if (Math.abs(dx) + Math.abs(dy) < d) {
          return;
        }
        sx = event.screenX;
        sy = event.screenY;
      }
      onDrag(event, dx, dy, dragging);
      dragging = true;
    };
    options = {
      capture: true
    };
    $.window.on("mousemove", onMouseMove, options);
    $.window.once("mouseup", (event) => {
      var dx, dy;
      $.window.off("mousemove", onMouseMove, options);
      dx = event.screenX - sx;
      dy = event.screenY - sy;
      onEnd(event, dx, dy, dragging);
    });
  };

  $.assign = function(a, b) {
    if (a === void 0 || a === null) {
      return b;
    }
    if (b === void 0 || b === null) {
      return a;
    }
    Object.keys(b).forEach((key) => {
      if (typeof b[key] === "object") {
        return $.assign(a[key], b[key]);
      } else {
        return a[key] = b[key];
      }
    });
    return a;
  };

  $.extend = function(a, b) {
    if (a === void 0 || a === null) {
      return b;
    }
    if (b === void 0 || b === null) {
      return a;
    }
    Object.keys(b).forEach((key) => {
      if (typeof b[key] === "object") {
        return $.extend(a[key], b[key]);
      } else {
        return a[key] = a[key] + " " + b[key];
      }
    });
    return a;
  };

  $.text = function(text) {
    return document.createTextNode(text);
  };

  $.$text = function(text) {
    return $(document.createTextNode(text));
  };

  $.createNS = function(ns, tag, attr, children) {
    return $.$createNS(ns, tag, attr, children).$;
  };

  $.$createNS = function(ns, tag, attr, children) {
    var elm;
    elm = $(document.createElementNS(ns, tag));
    if (attr) {
      elm.attr(attr);
    }
    if (children) {
      // elm.prop attr if attr
      elm.append(...children);
    }
    return elm;
  };

  $.create = function(tag, props, children) {
    return $.$create(tag, props, children).$;
  };

  $.$create = function(tag, props, children) {
    var elm, event, listeners, ref;
    elm = $(document.createElement(tag));
    if (props) {
      if ("src" in props) {
        console.warn("props.src should be moved to props.attr.src");
      }
      if ("className" in props) {
        elm.$.className = props.className;
      }
      if ("state" in props) {
        elm.val(props.state);
      }
      if ("props" in props) {
        elm.prop(props.props);
      }
      if ("attr" in props) {
        elm.attr(props.attr);
      }
      if ("style" in props) {
        elm.prop({
          style: props.style
        });
      }
      if ("on" in props) {
        ref = props.on;
        for (event in ref) {
          listeners = ref[event];
          elm.on(event, listeners);
        }
      }
    }
    if (children) {
      elm.append(...children);
    }
    return elm;
  };

  $.box = function(props, children) {
    return $.$box(props, children).$;
  };

  $.$box = function(props, children) {
    return $.$create("div", props, children);
  };

  //###############################################################################
  $.requireScript = function(src) {
    var pkg;
    console.warn("$.requireStyle is deprecated.");
    if ((src.startsWith("./")) || (src.startsWith("../"))) {
      pkg = document.currentScript.getAttribute("data-pkg");
      src = "/fs/lang/" + pkg + "/" + src;
    } else {
      if ((src.startsWith("http://")) || (src.startsWith("https://")) || (src.startsWith("/"))) {

      } else {
        // src = src
        src = "/fs/lang/" + src;
      }
    }
    return new Promise((resolve, reject) => {
      return $.$create("script", {
        attr: {
          src: src
        }
      }).on("error", (event) => {
        reject(event);
      }).on("load", (event) => {
        resolve(event);
      }).appendTo($.head);
    });
  };

  $.requireStyle = function(src) {
    var pkg;
    console.warn("$.requireStyle is deprecated. Use require.Style(..)\nSource: " + src);
    if ((src.startsWith("./")) || (src.startsWith("../"))) {
      pkg = document.currentScript.getAttribute("data-pkg");
      src = "/fs/lang/" + pkg + "/" + src;
    } else {
      if ((src.startsWith("http://")) || (src.startsWith("https://")) || (src.startsWith("/"))) {

      } else {
        // src = src
        src = "fs/lang/" + src;
      }
    }
    return new Promise((resolve, reject) => {
      return $.$create("link", {
        attr: {
          rel: "stylesheet",
          href: src
        }
      }).on("error", (event) => {
        reject(event);
      }).on("load", () => {
        resolve(event);
      }).appendTo($.head);
    });
  };

  //###############################################################################
  (function() {
    var counter, doc, style;
    counter = 0;
    style = $.create("style");
    style.title = "autogen styles";
    $.head.append(style);
    doc = style.sheet;
    Object.defineProperty(doc, "title", {
      value: "autogen styles"
    });
    $.style = (rule) => {
      var cls, r;
      cls = counter.toString(36);
      r = Object.keys(rule).map((key) => {
        var pseudo, rr;
        if (key[0] === ":") {
          pseudo = rule[key];
          rr = Object.keys(pseudo).map((key) => {
            return `${key}: ${pseudo[key]}`;
          }).join(";\n  ");
          doc.insertRule(`._${cls}${key} {\n  ${rr};\n}`);
          return "";
        }
        return `${key}: ${rule[key]}`;
      }).join(";\n  ");
      doc.insertRule(`._${cls} {\n  ${r};\n}`);
      counter++;
      return "_" + cls;
    };
    $.rawStyle = (rule) => {
      return doc.insertRule(rule);
    };
    return $.makeStyle = (states) => {
      var cls, rule, state, styles;
      cls = counter.toString(36);
      for (state in states) {
        styles = states[state];
        rule = Object.keys(styles).map((key) => {
          return `${key}: ${styles[key]}`;
        }).join(";\n  ");
        if (state === "default") {
          doc.insertRule(`._${cls} {  ${rule};\n}`);
        } else {
          doc.insertRule(`._${cls}${state} {  ${rule};\n}`);
        }
      }
      counter++;
      return "_" + cls;
    };
  })();

  //###############################################################################
  $.platform = function() {
    switch (false) {
      case !navigator.platform.startsWith("Win"):
        return "windows";
      case !navigator.platform.startsWith("Mac"):
        return "mac";
      case !navigator.platform.includes("Linux"):
        return "linux";
      default:
        return "";
    }
  };

  window.$ = $;

}).call(this);


//# sourceMappingURL=$.js.map
//# sourceURL=coffeescript