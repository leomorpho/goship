import { createApp } from "vue";
import Counter from "../javascript/vue/components/Counter.vue";

function normalizeProps(props = {}) {
  const initialCount = Number(props.initialCount);
  const localizedLabel =
    props?.i18n?.messages && typeof props.i18n.messages.label === "string"
      ? props.i18n.messages.label
      : "";
  return {
    initialCount: Number.isFinite(initialCount) ? initialCount : 0,
    label: localizedLabel || (typeof props.label === "string" ? props.label : "Vue Counter"),
  };
}

export function mount(el, props = {}) {
  createApp(Counter, normalizeProps(props)).mount(el);
}
