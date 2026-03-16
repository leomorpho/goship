import { createApp } from "vue";
import Counter from "../javascript/vue/components/Counter.vue";

function normalizeProps(props = {}) {
  const initialCount = Number(props.initialCount);
  return {
    initialCount: Number.isFinite(initialCount) ? initialCount : 0,
    label: typeof props.label === "string" ? props.label : "Vue Counter",
  };
}

export function mount(el, props = {}) {
  createApp(Counter, normalizeProps(props)).mount(el);
}
