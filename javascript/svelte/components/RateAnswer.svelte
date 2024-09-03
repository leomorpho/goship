<script>
  import { toast } from "wc-toast";
  import Slider from "./rateAnswer/Slider.svelte";

  export let answerId;
  export let effortValue = 50;
  export let clarityValue = 50;
  export let truthfulnessValue = 50;
  export let saveEndpoint = "/";
  export let csrf = "";

  let showModal = false;

  function toggleModal() {
    showModal = !showModal;
  }

  function showToast(message) {
    toast.error(message);
  }

  async function saveRating() {
    try {
      const response = await fetch(saveEndpoint + "?csrf" + csrf, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          "X-CSRF-Token": csrf,
        },
        body: JSON.stringify({
          effort: effortValue,
          clarity: clarityValue,
          truthfulness: truthfulnessValue,
        }),
      });

      if (!response.ok) {
        throw new Error("Failed to save rating");
      }
      showModal = false;
    } catch (error) {
      console.error("Error saving rating:", error);
      showToast(`${error.message || "Unknown error"}`);
    }
  }
</script>

<wc-toast></wc-toast>

<button
  on:click={toggleModal}
  class="bg-blue-500 text-white py-2 px-2 sm:px-4 rounded-full text-sm"
>
  Rate
</button>

{#if showModal}
  <div
    class="fixed inset-0 bg-black bg-opacity-50 dark:bg-opacity-80 flex justify-center items-center z-50"
  >
    <div
      class="bg-white dark:bg-slate-700 text-black dark:text-white p-6 rounded-lg w-full md:w-2/3 lg:w-1/2 relative m-5"
    >
      <button
        on:click={toggleModal}
        class="absolute top-2 right-2 text-red-400 dark:text-red-300 hover:text-red-500 dark:hover:text-red-500"
      >
        <svg
          xmlns="http://www.w3.org/2000/svg"
          viewBox="0 0 20 20"
          fill="currentColor"
          class="w-6 h-6"
        >
          <path
            fill-rule="evenodd"
            d="M10 18a8 8 0 1 0 0-16 8 8 0 0 0 0 16ZM8.28 7.22a.75.75 0 0 0-1.06 1.06L8.94 10l-1.72 1.72a.75.75 0 1 0 1.06 1.06L10 11.06l1.72 1.72a.75.75 0 1 0 1.06-1.06L11.06 10l1.72-1.72a.75.75 0 0 0-1.06-1.06L10 8.94 8.28 7.22Z"
            clip-rule="evenodd"
          />
        </svg>
      </button>
      <div class="m-3 sm:m-4 md:m-5">
        <h1 class="text-xl sm:text-2xl md:text-3xl">🤟 Rate Answer</h1>
      </div>
      <Slider
        sliderName="💪 Effort"
        startScaleName="Lazy"
        endScaleName="Effortful"
        bind:value={effortValue}
      />
      <Slider
        sliderName="💎 Clarity"
        startScaleName="Low"
        endScaleName="High"
        bind:value={clarityValue}
      />
      <Slider
        sliderName="✅ Truthfulness"
        startScaleName="Low"
        endScaleName="High"
        bind:value={truthfulnessValue}
      />
      <div class="flex justify-center items-center">
        <button
          type="button"
          class="m-3 focus:outline-none text-white bg-green-700 hover:bg-green-800 focus:ring-4 focus:ring-green-300 font-medium rounded-lg text-sm px-5 py-2.5 me-2 mb-2 dark:bg-green-600 dark:hover:bg-green-700 dark:focus:ring-green-800"
          on:click={saveRating}>Submit</button
        >
      </div>
    </div>
  </div>
{/if}
