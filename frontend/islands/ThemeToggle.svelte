<script>
  import { onMount, tick } from "svelte";

  let theme =
    localStorage.getItem("color-theme") ||
    (window.matchMedia("(prefers-color-scheme: dark)").matches
      ? "darkmode"
      : "lightmode");

  onMount(async () => {
    await tick();
    document.documentElement.setAttribute("data-theme", theme);
    document.documentElement.classList.toggle("dark", theme === "darkmode");
    updateIcons();
  });

  function toggleTheme() {
    theme = theme === "darkmode" ? "lightmode" : "darkmode";
    localStorage.setItem("color-theme", theme);
    document.documentElement.setAttribute("data-theme", theme);
    document.documentElement.classList.toggle("dark", theme === "darkmode");
    updateIcons();
  }

  function updateIcons() {
    theme = theme;
  }
</script>

<button
  on:click={toggleTheme}
  type="button"
  class="theme-toggle text-primary-content
    w-9 h-9 md:w-10 md:h-10 bg-slate-200 hover:bg-slate-300 dark:hover:bg-slate-600 dark:bg-slate-500 rounded-full flex items-center justify-center"
>
  {#if theme === "darkmode"}
    <div class="theme-toggle-dark-icon text-xl sm:text-2xl">🌞</div>
  {:else}
    <div class="theme-toggle-light-icon text-xl sm:text-2xl">🌚</div>
  {/if}
</button>
