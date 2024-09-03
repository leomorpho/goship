import { createCalHeatmap } from "./cal_heatmap";
import { loadScriptsAndStyles } from "./load_scripts_and_styles";

window.initializeJS = function initializeApp(targetElement) {
  console.log("initialize js");
  // Control zoom based on screen size
  controlZoom();

  // Set up PWA service worker
  if ("serviceWorker" in navigator) {
    navigator.serviceWorker
      .register("/service-worker.js", { type: "classic" })
      .then(function (registration) {
        console.log(
          "Service Worker registered with scope:",
          registration.scope
        );
      })
      .catch(function (error) {
        console.log("Service Worker registration failed:", error);
      });
  }

  refreshPageIfStale();
};

// controlZoom prevents zooming on mobile devices to improve user experience. Note that it is
// not enforced by all browsers, so is not guaranteed to work on all of them. TODO: we may want to
// remove that entirely as Brave/Chrome ignore these directives, and it may be globally useless if
// all browsers ignore it.
function controlZoom() {
  const viewportMeta = document.querySelector('meta[name="viewport"]');

  function updateViewport() {
    if (window.innerWidth < 1024) {
      // Disable zooming on small screens
      viewportMeta.setAttribute(
        "content",
        "width=device-width, initial-scale=1.0, maximum-scale=1.0, user-scalable=no"
      );
    } else {
      // Allow zooming on large screens
      viewportMeta.setAttribute(
        "content",
        "width=device-width, initial-scale=1.0"
      );
    }
  }

  // Update on initial load
  updateViewport();

  // Update on window resize
  window.addEventListener("resize", updateViewport);
}

function refreshPageIfStale() {
  document.addEventListener("visibilitychange", function () {
    if (document.visibilityState === "visible") {
      // Check the time elapsed
      const currentTime = new Date().getTime();
      const lastVisitTime = localStorage.getItem("lastVisitTime");
      const timeElapsed = currentTime - lastVisitTime;

      // Define a threshold (e.g., 10 minutes = 600000 ms)
      if (timeElapsed > 600000) {
        // 30 minutes
        console.log("Refreshing data after inactivity");
        window.location.reload();
      }
    } else {
      localStorage.setItem("lastVisitTime", new Date().getTime());
    }
  });
}

window.createCalHeatmap = createCalHeatmap;
window.loadScriptsAndStyles = loadScriptsAndStyles;
