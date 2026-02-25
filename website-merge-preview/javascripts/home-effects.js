(function () {
  "use strict";

  var scrollListenerBound = false;

  function random(min, max) {
    return Math.random() * (max - min) + min;
  }

  function markBodyState() {
    if (!document.body) return;
    var isHome = !!document.querySelector(".kubara-home");
    document.body.classList.toggle("kubara-home-page", isHome);
    document.body.classList.toggle("kubara-scrolled", window.scrollY > 24);
  }

  function fillParticles(container, count) {
    if (!container) return;
    container.textContent = "";
    var fragment = document.createDocumentFragment();

    for (var i = 0; i < count; i += 1) {
      var particle = document.createElement("span");
      particle.className = "kh-particle";
      particle.style.left = random(0, 100).toFixed(2) + "%";
      particle.style.top = random(0, 100).toFixed(2) + "%";
      particle.style.animationDelay = random(0, 6).toFixed(2) + "s";
      particle.style.animationDuration = random(10, 24).toFixed(2) + "s";
      fragment.appendChild(particle);
    }

    container.appendChild(fragment);
  }

  function fillStreams(container, count) {
    if (!container) return;
    container.textContent = "";
    var fragment = document.createDocumentFragment();

    for (var i = 0; i < count; i += 1) {
      var stream = document.createElement("span");
      stream.className = "kh-stream";
      stream.style.top = random(8, 92).toFixed(2) + "%";
      stream.style.left = random(0, 25).toFixed(2) + "%";
      stream.style.width = random(26, 54).toFixed(2) + "%";
      stream.style.animationDelay = random(0, 7).toFixed(2) + "s";
      stream.style.animationDuration = random(12, 20).toFixed(2) + "s";
      fragment.appendChild(stream);
    }

    container.appendChild(fragment);
  }

  function fillStars(container, count) {
    if (!container) return;
    container.textContent = "";
    var fragment = document.createDocumentFragment();

    for (var i = 0; i < count; i += 1) {
      var star = document.createElement("span");
      star.className = "kh-star";
      star.style.left = random(0, 100).toFixed(2) + "%";
      star.style.top = random(0, 100).toFixed(2) + "%";
      star.style.animationDelay = random(0, 3).toFixed(2) + "s";
      fragment.appendChild(star);
    }

    container.appendChild(fragment);
  }

  function ensureGlobalFx() {
    if (!document.body) return null;

    var fx = document.querySelector(".kubara-global-fx");
    if (fx) return fx;

    fx = document.createElement("div");
    fx.className = "kubara-global-fx";
    fx.setAttribute("aria-hidden", "true");
    fx.innerHTML =
      '<div class="kgfx-stars"></div>' +
      '<div class="kgfx-streams"></div>' +
      '<div class="kgfx-rings">' +
      '  <span class="kgfx-ring kgfx-ring-1"></span>' +
      '  <span class="kgfx-ring kgfx-ring-2"></span>' +
      '  <span class="kgfx-ring kgfx-ring-3"></span>' +
      '  <span class="kgfx-ring kgfx-ring-4"></span>' +
      '  <span class="kgfx-ring kgfx-ring-5"></span>' +
      "</div>";

    document.body.appendChild(fx);
    return fx;
  }

  function fillGlobalStars(container, count) {
    if (!container) return;
    container.textContent = "";

    var fragment = document.createDocumentFragment();
    for (var i = 0; i < count; i += 1) {
      var star = document.createElement("span");
      star.className = "kgfx-star";
      star.style.left = random(0, 100).toFixed(2) + "%";
      star.style.top = random(0, 100).toFixed(2) + "%";
      star.style.animationDelay = random(0, 4).toFixed(2) + "s";
      star.style.animationDuration = random(3, 6).toFixed(2) + "s";
      fragment.appendChild(star);
    }

    container.appendChild(fragment);
  }

  function fillGlobalStreams(container, count) {
    if (!container) return;
    container.textContent = "";

    var fragment = document.createDocumentFragment();
    for (var i = 0; i < count; i += 1) {
      var stream = document.createElement("span");
      stream.className = "kgfx-stream";
      stream.style.top = random(10, 88).toFixed(2) + "%";
      stream.style.left = random(-10, 15).toFixed(2) + "%";
      stream.style.width = random(18, 34).toFixed(2) + "%";
      stream.style.animationDelay = random(0, 9).toFixed(2) + "s";
      stream.style.animationDuration = random(18, 30).toFixed(2) + "s";
      fragment.appendChild(stream);
    }

    container.appendChild(fragment);
  }

  function initGlobalEffects(isHome) {
    var fx = ensureGlobalFx();
    if (!fx) return;

    fx.hidden = !!isHome;
    if (isHome) return;

    if (fx.dataset.kubaraInitialized === "1") return;

    fillGlobalStars(fx.querySelector(".kgfx-stars"), 18);
    fillGlobalStreams(fx.querySelector(".kgfx-streams"), 4);
    fx.dataset.kubaraInitialized = "1";
  }

  function initHomeEffects() {
    markBodyState();

    var home = document.querySelector(".kubara-home");
    if (home) {
      fillParticles(home.querySelector(".kh-particles"), 50);
      fillStreams(home.querySelector(".kh-streams"), 10);
      fillStars(home.querySelector(".kh-stars"), 22);
    }

    initGlobalEffects(!!home);

    if (!scrollListenerBound) {
      window.addEventListener(
        "scroll",
        function () {
          if (!document.body) return;
          document.body.classList.toggle("kubara-scrolled", window.scrollY > 24);
        },
        { passive: true }
      );
      scrollListenerBound = true;
    }
  }

  if (document.readyState === "loading") {
    document.addEventListener("DOMContentLoaded", initHomeEffects, { once: true });
  } else {
    initHomeEffects();
  }

  if (window.document$ && typeof window.document$.subscribe === "function") {
    window.document$.subscribe(function () {
      initHomeEffects();
    });
  }
})();
