const currentScript =
  document.currentScript instanceof HTMLScriptElement
    ? document.currentScript
    : null;

const manifestURL =
  currentScript?.dataset.manifestUrl || "/files/islands-manifest.json";

let manifestPromise;
const loadedStyleURLs = new Set();

async function getManifest() {
  if (!manifestPromise) {
    manifestPromise = fetch(manifestURL, { credentials: "same-origin" }).then(
      async (response) => {
        if (!response.ok) {
          throw new Error(
            `failed to load islands manifest (${response.status} ${response.statusText})`
          );
        }

        return response.json();
      }
    );
  }

  return manifestPromise;
}

async function mountIsland(el, manifest) {
  const islandName = el.dataset.island;
  if (!islandName) {
    return;
  }

  const islandEntry = manifest[islandName];
  const moduleURL =
    typeof islandEntry === "string" ? islandEntry : islandEntry?.script;
  if (!moduleURL) {
    console.warn(`Island "${islandName}" not found in manifest.`);
    return;
  }

  el.setAttribute("data-island-mounted", "true");

  try {
    const styleURLs =
      typeof islandEntry === "string" ? [] : islandEntry?.styles || [];
    for (const styleURL of styleURLs) {
      if (loadedStyleURLs.has(styleURL)) {
        continue;
      }
      loadedStyleURLs.add(styleURL);

      if (!document.querySelector(`link[data-island-style="${styleURL}"]`)) {
        const link = document.createElement("link");
        link.rel = "stylesheet";
        link.href = styleURL;
        link.dataset.islandStyle = styleURL;
        document.head.appendChild(link);
      }
    }

    const mod = await import(moduleURL);
    const props = JSON.parse(el.dataset.props || "{}");

    if (typeof mod.mount === "function") {
      await mod.mount(el, props);
      return;
    }

    if (typeof mod.default === "function") {
      new mod.default({ target: el, props });
      return;
    }

    throw new Error(`Island "${islandName}" is missing a mount(el, props) or default export.`);
  } catch (error) {
    el.removeAttribute("data-island-mounted");
    console.error(`Failed to mount island "${islandName}".`, error);
  }
}

async function mountPendingIslands(root = document) {
  const pending = root.querySelectorAll("[data-island]:not([data-island-mounted])");
  if (pending.length === 0) {
    return;
  }

  let manifest;
  try {
    manifest = await getManifest();
  } catch (error) {
    console.error("Failed to load islands manifest.", error);
    return;
  }

  await Promise.all(Array.from(pending, (el) => mountIsland(el, manifest)));
}

document.addEventListener("DOMContentLoaded", () => {
  void mountPendingIslands();
});

document.addEventListener("htmx:afterSettle", () => {
  void mountPendingIslands();
});

if (document.readyState !== "loading") {
  void mountPendingIslands();
}
