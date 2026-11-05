// Small enhancements over the rendered pages; every page reads fully without them.
(function () {
  var site = window.TFUI || { base: "" };

  function anchors() {
    document.querySelectorAll(".prose h2[id], .prose h3[id], .prose h4[id]").forEach(function (h) {
      var a = document.createElement("a");
      a.className = "anchor";
      a.href = "#" + h.id;
      a.setAttribute("aria-label", "Link to this section");
      a.textContent = "#";
      h.insertBefore(a, h.firstChild);
    });
  }

  function toc() {
    var aside = document.querySelector(".toc");
    var headings = document.querySelectorAll(".prose h2[id]");
    if (!aside || headings.length < 3) return;
    var title = document.createElement("p");
    title.textContent = "On this page";
    var list = document.createElement("ul");
    var links = [];
    headings.forEach(function (h) {
      var li = document.createElement("li");
      var a = document.createElement("a");
      a.href = "#" + h.id;
      a.textContent = h.textContent.replace(/^#/, "");
      li.appendChild(a);
      list.appendChild(li);
      links.push([h, a]);
    });
    aside.appendChild(title);
    aside.appendChild(list);
    if (!("IntersectionObserver" in window)) return;
    var observer = new IntersectionObserver(function () {
      var current = links[0];
      links.forEach(function (pair) {
        if (pair[0].getBoundingClientRect().top < 140) current = pair;
      });
      links.forEach(function (pair) { pair[1].classList.toggle("active", pair === current); });
    }, { rootMargin: "0px 0px -60% 0px", threshold: [0, 1] });
    headings.forEach(function (h) { observer.observe(h); });
  }

  function copyButton(text) {
    var button = document.createElement("button");
    button.type = "button";
    button.className = "copy";
    button.textContent = "copy";
    button.addEventListener("click", function () {
      navigator.clipboard.writeText(text()).then(function () {
        button.textContent = "copied";
        setTimeout(function () { button.textContent = "copy"; }, 1500);
      });
    });
    return button;
  }

  function copyButtons() {
    if (!navigator.clipboard) return;
    document.querySelectorAll("div.highlighter-rouge, figure.highlight").forEach(function (block) {
      var code = block.querySelector("pre code") || block.querySelector("pre");
      if (!code) return;
      block.appendChild(copyButton(function () { return code.textContent; }));
    });
    document.querySelectorAll(".install-line").forEach(function (line) {
      var cmd = line.querySelector("code");
      line.appendChild(copyButton(function () { return cmd.textContent; }));
    });
  }

  function search() {
    var el = document.getElementById("search");
    if (!el || !window.PagefindUI) return;
    new window.PagefindUI({
      element: "#search",
      baseUrl: site.base + "/",
      showImages: false,
      showSubResults: true,
      resetStyles: false,
      translations: { placeholder: "Search the docs" }
    });
  }

  // On the landing page's wide layout, a wheel notch or arrow key at a section's edge moves a whole section.
  // Mandatory snap alone sends a short scroll back to the edge it started from, so the reader would never leave it.
  function slides() {
    var snaps = Array.prototype.slice.call(document.querySelectorAll(".snap"));
    var footer = document.querySelector(".footer");
    var wide = window.matchMedia("(min-width: 981px)");
    if (!snaps.length || !footer) return;
    snaps.push(footer);
    var busy = false;

    function line() { return document.querySelector(".topbar").getBoundingClientRect().bottom; }
    function go(top) {
      busy = true;
      window.scrollTo({ top: top, behavior: "smooth" });
      setTimeout(function () { busy = false; }, 750);
    }
    // The scroll offset that pages from the section at the snap line to its neighbour, or null to scroll natively.
    function target(dir) {
      var edge = line(), vh = window.innerHeight, y = window.scrollY;
      var i = 0;
      while (i < snaps.length - 1 && snaps[i + 1].getBoundingClientRect().top <= edge + 2) i++;
      var box = snaps[i].getBoundingClientRect();
      if (dir > 0) {
        if (i === snaps.length - 1 || box.bottom > vh + 2) return null;
        var next = snaps[i + 1];
        return next === footer ? document.documentElement.scrollHeight - vh : y + next.getBoundingClientRect().top - edge;
      }
      if (i === 0 || box.top < edge - 2) return null;
      var prev = snaps[i - 1].getBoundingClientRect();
      return prev.height <= vh - edge ? y + prev.top - edge : y + prev.bottom - vh;
    }
    function page(ev, dir) {
      if (!wide.matches) return;
      if (busy) { ev.preventDefault(); return; }
      var top = target(dir);
      if (top === null) return;
      ev.preventDefault();
      go(top);
    }
    window.addEventListener("wheel", function (ev) {
      if (ev.ctrlKey || Math.abs(ev.deltaY) < Math.abs(ev.deltaX)) return;
      page(ev, ev.deltaY > 0 ? 1 : -1);
    }, { passive: false });
    document.addEventListener("keydown", function (ev) {
      if (ev.defaultPrevented || ev.altKey || ev.ctrlKey || ev.metaKey) return;
      if (ev.target.closest && ev.target.closest("input, textarea, select, pre[tabindex], [contenteditable]")) return;
      var dir = { ArrowDown: 1, PageDown: 1, " ": ev.shiftKey ? -1 : 1, ArrowUp: -1, PageUp: -1 }[ev.key];
      if (dir) page(ev, dir);
    });
  }

  anchors();
  toc();
  copyButtons();
  search();
  slides();
})();
