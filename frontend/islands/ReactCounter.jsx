import { useState } from "react";
import { createRoot } from "react-dom/client";

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

function Counter({ initialCount, label }) {
  const [count, setCount] = useState(toInitialCount(initialCount));

  return (
    <section
      data-component="counter-react"
      className="rounded-xl border border-base-300 bg-base-100 p-4 space-y-3"
    >
      <h3 className="text-lg font-semibold">{label || "React Counter"}</h3>
      <p data-slot="count" className="text-3xl font-bold leading-none">{count}</p>
      <p className="mt-2 text-xs font-medium uppercase tracking-wide opacity-70">React</p>
      <div className="flex gap-2">
        <button type="button" className="btn btn-sm" onClick={() => setCount((v) => v - 1)}>
          -1
        </button>
        <button type="button" className="btn btn-sm" onClick={() => setCount((v) => v + 1)}>
          +1
        </button>
      </div>
    </section>
  );
}

export function mount(el, props = {}) {
  const root = createRoot(el);
  root.render(<Counter initialCount={props.initialCount} label={resolveLabel(props, "React Counter")} />);
}
