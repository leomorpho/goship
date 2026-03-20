<template>
  <section
    data-component="counter-vue"
    class="rounded-xl border border-base-300 bg-base-100 p-4 space-y-3"
  >
    <h3 class="text-lg font-semibold">{{ labelToShow }}</h3>
    <p data-slot="count" class="text-3xl font-bold leading-none">{{ count }}</p>
    <p class="mt-2 text-xs font-medium uppercase tracking-wide opacity-70">Vue</p>
    <div class="flex gap-2">
      <button type="button" class="btn btn-sm" @click="decrement">-1</button>
      <button type="button" class="btn btn-sm" @click="increment">+1</button>
    </div>
  </section>
</template>

<script setup>
import { computed, ref } from "vue";

const props = defineProps({
  initialCount: {
    type: Number,
    default: 0,
  },
  label: {
    type: String,
    default: "",
  },
});

const count = ref(Number.isFinite(props.initialCount) ? props.initialCount : 0);
const labelToShow = computed(() => props.label || "Vue Counter");

const decrement = () => {
  count.value -= 1;
};

const increment = () => {
  count.value += 1;
};
</script>
