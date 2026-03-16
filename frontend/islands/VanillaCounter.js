function toInitialCount(value) {
  const parsed = Number(value);
  if (Number.isFinite(parsed)) {
    return parsed;
  }
  return 0;
}

export function mount(el, props = {}) {
  const state = {
    count: toInitialCount(props.initialCount),
    label: typeof props.label === "string" && props.label.length > 0
      ? props.label
      : "Vanilla Counter",
  };

  const container = document.createElement("section");
  container.dataset.component = "counter-vanilla";
  container.className = "rounded-xl border border-base-300 bg-base-100 p-4 space-y-3";

  const title = document.createElement("h3");
  title.className = "text-lg font-semibold";
  title.textContent = state.label;

  const value = document.createElement("p");
  value.className = "text-3xl font-bold leading-none";
  value.setAttribute("data-slot", "count");

  const framework = document.createElement("p");
  framework.className = "mt-2 text-xs font-medium uppercase tracking-wide opacity-70";
  framework.textContent = "Vanilla JS";

  const controls = document.createElement("div");
  controls.className = "flex gap-2";

  const decrement = document.createElement("button");
  decrement.type = "button";
  decrement.className = "btn btn-sm";
  decrement.textContent = "-1";

  const increment = document.createElement("button");
  increment.type = "button";
  increment.className = "btn btn-sm";
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
