// The landing page's "See it work" tour: a small simulation of tfui's plan view, fed by docs/_data/tour.json (one
// real plan of demo/before-after). Six scenes play on their own; focusing the terminal hands the keys to the reader.
// Without JavaScript the stage keeps its static list, so the page reads complete either way.
(function () {
  var root = document.getElementById("how-it-works");
  var dataEl = document.getElementById("tour-data");
  if (!root || !dataEl) return;

  var data = JSON.parse(dataEl.textContent);
  var stage = root.querySelector(".stage pre");
  var title = root.querySelector(".stage .shot-bar b");
  var live = root.querySelector(".stage .live");
  var caption = root.querySelector(".tour-caption");
  var tabs = Array.prototype.slice.call(root.querySelectorAll(".tour-steps button"));
  var replay = root.querySelector(".tour-replay");
  var reduced = window.matchMedia && window.matchMedia("(prefers-reduced-motion: reduce)").matches;

  function lines(id) {
    var tpl = document.getElementById(id);
    return tpl ? tpl.innerHTML.replace(/\{% (end)?raw %\}/g, "").replace(/\n$/, "").split("\n") : [];
  }
  var wall = lines("tour-before");
  var agent = lines("tour-agent");

  var SYM = { create: "+", update: "~", delete: "-" };
  var CLS = { create: "add", update: "chg", delete: "del" };
  var COLS = 56;
  var ROWS = 20;

  function esc(s) { return String(s).replace(/&/g, "&amp;").replace(/</g, "&lt;").replace(/>/g, "&gt;"); }
  function span(cls, text) { return '<span class="' + cls + '">' + esc(text) + "</span>"; }
  function badge(risk) { return span("r-" + risk.toLowerCase(), "[" + risk + "]"); }
  function sym(change) { return span(CLS[change.action], SYM[change.action]); }

  // ---- model -------------------------------------------------------------------------------------------------
  var state;
  function reset() {
    state = { frame: "list", tree: true, filter: "", typing: false, cursor: 0, pins: {}, collapsed: {},
      inspect: null, popped: null, out: [] };
  }

  function matches(change) {
    if (!state.filter) return true;
    var q = state.filter.toLowerCase(), s = change.address.toLowerCase(), i = 0;
    for (var j = 0; j < s.length && i < q.length; j++) if (s[j] === q[i]) i++;
    return i === q.length;
  }

  // The rows the cursor walks: module branches and their changes in tree mode, bare changes in flat mode.
  function entries() {
    var shown = data.changes.filter(matches);
    if (!state.tree) return shown.map(function (c) { return { change: c }; });
    var out = [], modules = [];
    shown.forEach(function (c) { if (modules.indexOf(c.module) < 0) modules.push(c.module); });
    modules.forEach(function (m) {
      var kids = shown.filter(function (c) { return c.module === m; });
      out.push({ module: m, kids: kids });
      if (state.collapsed[m]) return;
      kids.forEach(function (c, i) { out.push({ change: c, last: i === kids.length - 1 }); });
    });
    return out;
  }

  function pinned() { return data.changes.filter(function (c) { return state.pins[c.address]; }); }

  function box(entry) {
    if (entry.change) return state.pins[entry.change.address] ? span("pinned", "[*]") : "[ ]";
    var n = entry.kids.filter(function (c) { return state.pins[c.address]; }).length;
    if (n === 0) return "[ ]";
    return n === entry.kids.length ? span("pinned", "[*]") : span("part", "[-]");
  }

  // ---- frames ------------------------------------------------------------------------------------------------
  function listFrame() {
    var list = entries();
    if (state.cursor >= list.length) state.cursor = Math.max(0, list.length - 1);
    var head = " Plan (" + data.changes.length + ")";
    var tail = list.length ? "[" + (state.cursor + 1) + "/" + list.length + "] " : "";
    var filter = state.typing ? "  / " + esc(state.filter) + '<span class="caret">█</span>'
      : state.filter ? '  <span class="flt">filter: ' + esc(state.filter) + "</span>" : "";
    var plain = head + (state.typing ? "  / " + state.filter + "█" : state.filter ? "  filter: " + state.filter : "");
    var out = [esc(head) + filter + " ".repeat(Math.max(1, COLS - plain.length - tail.length)) + span("meta", tail)];
    if (!list.length) out.push(span("meta", " No matching changes."));
    list.forEach(function (e, i) {
      var text;
      if (e.module) {
        text = box(e) + " " + span("meta", state.collapsed[e.module] ? "▶" : "▼") + " " + esc(e.module) + span("meta", " (" + e.kids.length + ")");
      } else if (state.tree && e.change.module) {
        text = box(e) + " " + span("meta", e.last ? "└─" : "├─") + sym(e.change) + " " + esc(e.change.name) + " " + badge(e.change.risk);
      } else {
        text = box(e) + " " + sym(e.change) + " " + esc(e.change.address) + " " + badge(e.change.risk);
      }
      var cls = "ln" + (i === state.cursor ? " sel" : "") + (e.change && e.change.address === state.popped ? " pop" : "")
        + (e.change && e.change.risk === "CRITICAL" ? " hot" : "");
      out.push({ cls: cls, html: " " + text });
    });
    var s = data.summary;
    out.push("", ' <span class="ok">Plan: ' + s.add + " to add, " + s.change + " to change, " + s.destroy + " to destroy</span>",
      " Overall risk: " + badge(data.risk).replace("[" + data.risk + "]", data.risk));
    var n = pinned().length;
    var hint = state.typing ? " enter keep  esc clear"
      : n ? " / filter  enter inspect  a apply" : " / filter  enter inspect  space pin  a apply";
    return { title: "tfui plan", body: out, foot: span("meta", hint) + (n ? "  " + span("chip", n + " pinned") : "") };
  }

  function inspectFrame() {
    var c = state.inspect;
    var out = [" " + esc(c.address) + (state.pins[c.address] ? " " + span("pinned", "[pinned]") : ""), "",
      span("meta", " Action:  ") + sym(c) + " " + c.action,
      span("meta", " Module:  ") + esc(c.module || "(root)"),
      span("meta", " Type:    ") + esc(c.type),
      span("meta", " Risk:    ") + badge(c.risk), "", span("meta", " Attributes:")];
    c.attrs.forEach(function (a) {
      if (c.action === "update") {
        out.push("   " + esc(a.key) + ":", "     " + span("del", "- " + a.before), "     " + span("add", "+ " + a.after));
      } else {
        var v = c.action === "create" ? span("add", "+ " + a.after) : span("del", "- " + a.before);
        out.push("   " + esc(a.key) + " " + v);
      }
    });
    return { title: "tfui plan · inspect", body: out, foot: span("meta", " esc back  space pin") };
  }

  function targets() { var p = pinned(); return p.length ? p : data.changes; }

  function confirmFrame() {
    var t = targets();
    var add = t.filter(function (c) { return c.action === "create"; }).length;
    var chg = t.filter(function (c) { return c.action === "update"; }).length;
    var del = t.filter(function (c) { return c.action === "delete"; }).length;
    var out = [" Apply " + t.length + (pinned().length ? " pinned" : "") + " change" + (t.length === 1 ? "" : "s") + "?", ""];
    t.forEach(function (c) { out.push("   " + sym(c) + " " + esc(c.address)); });
    out.push("", ' <span class="ok">Plan: ' + add + " to add, " + chg + " to change, " + del + " to destroy</span>");
    if (pinned().length) out.push(span("meta", " Planned with -target. Nothing else is touched."));
    return { title: "tfui apply", body: out, foot: span("meta", " y apply  n cancel") };
  }

  function outFrame(titleText, foot) {
    return { title: titleText, body: state.out.slice(), foot: foot ? span("meta", foot) : "" };
  }

  function render() {
    var f = state.frame === "list" ? listFrame()
      : state.frame === "inspect" ? inspectFrame()
      : state.frame === "confirm" ? confirmFrame()
      : state.frame === "wall" ? { title: "terraform plan", body: state.out.slice(), foot: "", wall: true }
      : state.frame === "shell" ? outFrame("agent session")
      : outFrame("tfui apply", " esc back to the plan");
    var body = f.body.slice(0, ROWS - 1);
    var html = body.map(function (l) {
      return typeof l === "string" ? '<span class="ln">' + (l || " ") + "</span>" : '<span class="' + l.cls + '">' + l.html + "</span>";
    });
    while (html.length < ROWS - 1) html.push('<span class="ln"> </span>');
    html.push('<span class="ln foot">' + (f.foot || " ") + "</span>");
    if (stage.dataset.frame !== state.frame) {
      stage.dataset.frame = state.frame;
      stage.classList.remove("swap");
      void stage.offsetWidth;
      stage.classList.add("swap");
    }
    stage.classList.toggle("wall", !!f.wall);
    stage.innerHTML = html.join("");
    title.textContent = f.title;
    state.popped = null;
  }

  // ---- keys --------------------------------------------------------------------------------------------------
  function move(by) {
    var n = entries().length;
    state.cursor = Math.min(Math.max(0, state.cursor + by), Math.max(0, n - 1));
  }

  function pin() {
    var e = entries()[state.cursor];
    if (!e) return;
    var group = e.change ? [e.change] : e.kids;
    var all = group.every(function (c) { return state.pins[c.address]; });
    group.forEach(function (c) { if (all) delete state.pins[c.address]; else state.pins[c.address] = true; });
    if (e.change && !all) state.popped = e.change.address;
  }

  function handle(key) {
    var f = state.frame;
    if (f === "list" && state.typing) {
      if (key === "Enter") state.typing = false;
      else if (key === "Escape") { state.typing = false; state.filter = ""; }
      else if (key === "Backspace") state.filter = state.filter.slice(0, -1);
      else if (key.length === 1 && key !== " ") { state.filter += key; state.cursor = 0; }
      else return false;
    } else if (f === "list") {
      var e = entries()[state.cursor];
      if (key === "j" || key === "ArrowDown") move(1);
      else if (key === "k" || key === "ArrowUp") move(-1);
      else if (key === "g" || key === "Home") state.cursor = 0;
      else if (key === "G" || key === "End") state.cursor = entries().length - 1;
      else if (key === "/") { state.typing = true; state.filter = ""; state.cursor = 0; }
      else if (key === "Escape") { state.filter = ""; }
      else if (key === " ") pin();
      else if (key === "^u") state.pins = {};
      else if (key === "^t") { state.tree = !state.tree; state.cursor = 0; }
      else if (key === "[") { data.changes.forEach(function (c) { state.collapsed[c.module] = true; }); state.cursor = 0; }
      else if (key === "]") state.collapsed = {};
      else if ((key === "Enter" || key === "i") && e && e.module) state.collapsed[e.module] = !state.collapsed[e.module];
      else if ((key === "Enter" || key === "i") && e) { state.inspect = e.change; state.frame = "inspect"; }
      else if (key === "a") state.frame = "confirm";
      else return false;
    } else if (f === "inspect") {
      if (key === "Escape" || key === "q") state.frame = "list";
      else if (key === " ") { var a = state.inspect.address; if (state.pins[a]) delete state.pins[a]; else state.pins[a] = true; }
      else return false;
    } else if (f === "confirm") {
      if (key === "y" || key === "Enter") return applyNow(), true;
      if (key === "n" || key === "Escape") state.frame = "list";
      else return false;
    } else if (f === "applied") {
      if (key === "Escape" || key === "Backspace" || key === "q") { state.frame = "list"; state.pins = {}; }
      else return false;
    } else return false;
    render();
    return true;
  }

  function applyLines(t) {
    var out = [];
    t.forEach(function (c) {
      var name = c.name.length > 34 ? c.name.slice(0, 33) + "…" : c.name;
      var verb = { create: ["Creating...", "Creation complete"], update: ["Modifying...", "Modifications complete"],
        delete: ["Destroying...", "Destruction complete"] }[c.action];
      out.push([" " + esc(name) + ": " + span("meta", verb[0]), " " + esc(name) + ": " + span("add", verb[1])]);
    });
    return out;
  }

  function summaryLine(t) {
    var n = function (a) { return t.filter(function (c) { return c.action === a; }).length; };
    return ' <span class="ok">Apply complete. ' + n("create") + " added, " + n("update") + " changed, " + n("delete") + " destroyed.</span>";
  }

  // In live mode the apply plays out at once; the scripted scene paces the same lines.
  function applyNow() {
    var t = targets();
    state.out = [];
    applyLines(t).forEach(function (pair) { state.out.push(pair[0], pair[1]); });
    state.out.push("", summaryLine(t));
    state.frame = "applied";
    render();
  }

  // ---- player ------------------------------------------------------------------------------------------------
  var run = 0;
  function Cancelled() {}
  function wait(ms, id) {
    return new Promise(function (resolve, reject) {
      if (id !== run) return reject(new Cancelled());
      if (reduced || !ms) return resolve();
      setTimeout(function () { id === run ? resolve() : reject(new Cancelled()); }, ms);
    });
  }
  function find(address) {
    var list = entries();
    for (var i = 0; i < list.length; i++) if (list[i].change && list[i].change.address === address) return i;
    return 0;
  }
  function byAction(action) { return data.changes.filter(function (c) { return c.action === action; }); }
  var bucket = data.changes.filter(function (c) { return c.type === "aws_s3_bucket"; })[0] || byAction("update")[0];
  var created = data.changes.filter(function (c) { return c.module === bucket.module && c.action === "create"; })[0] || byAction("create")[0];
  var word = bucket.module.replace(/^module\./, "");

  async function typeInto(text, id) {
    for (var i = 0; i < text.length; i++) { handle(text[i]); await wait(110, id); }
  }
  async function walkTo(address, id) {
    var target = find(address);
    while (state.cursor !== target) { handle(state.cursor < target ? "j" : "k"); await wait(170, id); }
  }

  var scenes = [
    { say: "<kbd>tfui plan</kbd> " + data.changes.length + " changes, sorted into modules, a risk badge on every one. Not " + wall.length + " lines.",
      ms: 7000,
      play: async function (id) {
        state.frame = "wall";
        for (var i = 0; i + ROWS - 1 <= wall.length; i += 7) { state.out = wall.slice(i, i + ROWS - 1); render(); await wait(28, id); }
        await wait(300, id);
        state.frame = "list"; state.tree = false; render();
        await wait(1500, id);
        handle("^t");
        await wait(800, id);
        state.cursor = find(byAction("delete")[0].address); render();
      } },
    { say: "<kbd>/</kbd> Type to narrow the plan. The tree keeps its shape.", ms: 5200,
      play: async function (id) {
        await wait(500, id);
        handle("/"); await wait(400, id);
        await typeInto(word, id); await wait(500, id);
        handle("Enter");
      } },
    { say: "<kbd>enter</kbd> Read one change's attribute diff. Nothing to scroll back through.", ms: 5600,
      before: function () { state.filter = word; },
      play: async function (id) {
        await wait(400, id);
        await walkTo(bucket.address, id); await wait(500, id);
        handle("Enter");
      } },
    { say: "<kbd>space</kbd> Pin the changes you mean to apply. Leave the rest alone.", ms: 6000,
      before: function () { state.filter = word; state.cursor = find(bucket.address); },
      play: async function (id) {
        await wait(400, id);
        await walkTo(created.address, id); await wait(300, id);
        handle(" "); await wait(700, id);
        await walkTo(bucket.address, id); await wait(300, id);
        handle(" ");
      } },
    { say: "<kbd>a</kbd> Apply only the pins: a targeted plan, one confirmation.", ms: 7600,
      before: function () { state.pins[created.address] = true; state.pins[bucket.address] = true; state.cursor = find(bucket.address); },
      play: async function (id) {
        await wait(600, id);
        state.frame = "out";
        state.out = [span("meta", " Planning with " + pinned().length + " targets..."), ""].concat(pinned().map(function (c) {
          return span("meta", "   -target=") + esc(c.address);
        }));
        render(); await wait(1200, id);
        state.frame = "confirm"; render(); await wait(1800, id);
        var t = targets();
        state.out = []; state.frame = "applied";
        var steps = applyLines(t);
        for (var i = 0; i < steps.length; i++) {
          state.out.push(steps[i][0]); render(); await wait(500, id);
          state.out.push(steps[i][1]); render(); await wait(250, id);
        }
        state.out.push("", summaryLine(t)); render();
      } },
    { say: "<kbd>--ci</kbd> Your agent gets " + agent.length + " lines, the risk, and exit code 2: changes present.", ms: 6400,
      play: async function (id) {
        state.frame = "shell"; state.out = [""]; render();
        var cmd = "tfui plan";
        for (var i = 0; i <= cmd.length; i++) { state.out[0] = ' <span class="prompt">$</span> ' + cmd.slice(0, i); render(); await wait(70, id); }
        await wait(400, id);
        for (var j = 0; j < agent.length; j++) { state.out.push(" " + agent[j].replace(/<span class="row"[^>]*>|<\/span>$/g, "")); render(); await wait(90, id); }
        await wait(500, id);
        state.out.push(' <span class="prompt">$</span> echo $?'); render(); await wait(500, id);
        state.out.push(' <span class="ok">2</span>'); render();
      } }
  ];

  var current = 0, auto = false, visible = false;

  function select(n) {
    tabs.forEach(function (t, i) {
      t.setAttribute("aria-selected", i === n ? "true" : "false");
      t.parentNode.classList.toggle("done", i < n);
      var bar = t.parentNode.querySelector(".bar");
      bar.style.transition = "none";
      bar.style.transform = i < n ? "scaleX(1)" : "scaleX(0)";
    });
    caption.innerHTML = scenes[n].say;
  }

  async function go(n, keepPlaying) {
    var id = ++run;
    current = n;
    auto = keepPlaying !== false && !reduced;
    live.hidden = true;
    reset();
    // Each scene sets up the state the previous one ended in, so a tab click lands mid-story.
    if (scenes[n].before) scenes[n].before();
    render();
    select(n);
    if (auto) {
      var bar = tabs[n].parentNode.querySelector(".bar");
      void bar.offsetWidth;
      bar.style.transition = "transform " + scenes[n].ms + "ms linear";
      bar.style.transform = "scaleX(1)";
    }
    var started = Date.now();
    try {
      await scenes[n].play(id);
      if (!auto) return;
      await wait(Math.max(600, scenes[n].ms - (Date.now() - started)), id);
      if (visible) go((n + 1) % scenes.length);
      else auto = false;
    } catch (err) {
      if (!(err instanceof Cancelled)) throw err;
    }
  }

  function takeOver() {
    if (live.hidden) {
      run++;
      auto = false;
      live.hidden = false;
      tabs.forEach(function (t) { var bar = t.parentNode.querySelector(".bar"); bar.style.transition = "none"; });
      if (state.frame === "wall" || state.frame === "shell" || state.frame === "out") { reset(); render(); }
      caption.innerHTML = "<kbd>you</kbd> Try <kbd>/</kbd> <kbd>j</kbd> <kbd>k</kbd> <kbd>enter</kbd> <kbd>space</kbd> <kbd>a</kbd> <kbd>[</kbd> <kbd>]</kbd> <kbd>esc</kbd>.";
    }
  }

  stage.addEventListener("keydown", function (ev) {
    if (ev.metaKey || ev.altKey) return;
    var key = ev.ctrlKey ? (ev.key.length === 1 ? "^" + ev.key.toLowerCase() : "") : ev.key;
    if (!key || key === "Tab" || key === "Shift") return;
    takeOver();
    if (handle(key)) ev.preventDefault();
  });
  stage.addEventListener("pointerdown", function () { stage.focus(); });
  tabs.forEach(function (t, i) { t.addEventListener("click", function () { go(i); }); });
  if (replay) replay.addEventListener("click", function () { go(0); });

  root.classList.add("js");
  window.tfuiTour = { go: function (n) { return go(n, false); }, handle: handle };
  reset();
  if (reduced) { go(0, false); return; }
  render();
  select(0);
  if (!("IntersectionObserver" in window)) { visible = true; go(0); return; }
  new IntersectionObserver(function (items) {
    visible = items[0].isIntersecting;
    if (visible && !auto && live.hidden) go(current);
    if (!visible && auto) { run++; auto = false; }
  }, { threshold: 0.35 }).observe(stage);

})();
