<script>
  import { createEventDispatcher } from "svelte";

  const dispatch = createEventDispatcher();
  export let permissionGranted = false;
  export let disabled = false;
</script>

<button
  {disabled}
  type="button"
  on:click={() => dispatch("click")}
  class="p-2 m-2 text-sm font-medium focus:outline-none rounded-full {permissionGranted
    ? 'text-blue-900 bg-blue-400 hover:bg-blue-600 hover:text-blue-900 dark:bg-blue-700 dark:text-blue-300 dark:hover:text-white dark:hover:bg-blue-700'
    : 'text-gray-600 bg-slate-200 hover:bg-blue-200 hover:text-blue-800 dark:bg-gray-700 dark:text-gray-400 dark:hover:text-white dark:hover:bg-blue-700'}"
>
  <slot name="icon" />
</button>
