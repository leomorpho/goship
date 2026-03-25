function toInitialCount(value) {
  const parsed = Number(value);
  if (Number.isFinite(parsed)) {
    return parsed;
  }
  return 0;
}

function resolveLabel(props = {}, fallback) {
  if (props?.i18n?.messages && typeof props.i18n.messages.label === "string" && props.i18n.messages.label.length > 0) {
    return props.i18n.messages.label;
  }
  if (typeof props.label === "string" && props.label.length > 0) {
    return props.label;
  }
  return fallback;
}

export function mount(el, props = {}) {
  const state = {
    count: toInitialCount(props.initialCount),
    label: resolveLabel(props, "Vanilla Counter"),
  };

  const container = document.createElement("section");
  container.dataset.component = "counter-vanilla";
  container.className = "gs-card gs-stack";

  const title = document.createElement("h3");
  title.className = "gs-title";
  title.textContent = state.label;

  const value = document.createElement("p");
  value.className = "gs-text";
  value.setAttribute("data-slot", "count");

  const framework = document.createElement("p");
  framework.className = "gs-kicker";
  framework.textContent = "Vanilla JS";

  const controls = document.createElement("div");
  controls.className = "gs-nav";

  const decrement = document.createElement("button");
  decrement.type = "button";
  decrement.className = "gs-button gs-button-secondary";
  decrement.textContent = "-1";

  const increment = document.createElement("button");
  increment.type = "button";
  increment.className = "gs-button gs-button-primary";
  increment.textContent = "+1";

  const render = () => {
    value.textContent = String(state.count);
  };

  decrement.addEventListener("click", () => {
    state.count -= 1;
    render();
  });
  increment.addEventListener("click", () => {
    state.count += 1;
    render();
  });

  controls.appendChild(decrement);
  controls.appendChild(increment);
  container.appendChild(title);
  container.appendChild(value);
  container.appendChild(framework);
  container.appendChild(controls);

  el.replaceChildren(container);
  render();
}
