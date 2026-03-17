<script>
  export let initialCount = 0;
  export let label = "Svelte Counter";
  export let i18n = {};

  let count = Number.isFinite(Number(initialCount)) ? Number(initialCount) : 0;
  $: localizedLabel = i18n?.messages?.label || label || "Svelte Counter";

  const decrement = () => {
    count -= 1;
  };

  const increment = () => {
    count += 1;
  };
</script>

<section
  data-component="counter-svelte"
  class="rounded-xl border border-base-300 bg-base-100 p-4 space-y-3"
>
  <h3 class="text-lg font-semibold">{localizedLabel}</h3>
  <p data-slot="count" class="text-3xl font-bold leading-none">{count}</p>
  <p class="mt-2 text-xs font-medium uppercase tracking-wide opacity-70">Svelte</p>
  <div class="flex gap-2">
    <button type="button" class="btn btn-sm" on:click={decrement}>-1</button>
    <button type="button" class="btn btn-sm" on:click={increment}>+1</button>
  </div>
</section>
