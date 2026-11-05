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

  // On the landing page, a pin on the base and its legend entry light up together.
  function basePins() {
    document.querySelectorAll("[data-pin]").forEach(function (el) {
      var id = el.getAttribute("data-pin");
      var pair = document.querySelectorAll('[data-pin="' + id + '"]');
      function set(on) { pair.forEach(function (p) { p.classList.toggle("on", on); }); }
      el.addEventListener("mouseenter", function () { set(true); });
      el.addEventListener("mouseleave", function () { set(false); });
      el.addEventListener("focusin", function () { set(true); });
      el.addEventListener("focusout", function () { set(false); });
    });
  }

  // The how-it-works flow animates once it scrolls into view; without JS it stays a complete static picture.
  function flow() {
    var el = document.querySelector(".flow");
    if (!el || !("IntersectionObserver" in window)) return;
    var observer = new IntersectionObserver(function (entries) {
      if (!entries[0].isIntersecting) return;
      el.classList.add("play");
      observer.disconnect();
    }, { threshold: 0.25 });
    observer.observe(el);
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

  anchors();
  toc();
  copyButtons();
  basePins();
  flow();
  search();
})();
