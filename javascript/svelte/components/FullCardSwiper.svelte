<script lang="ts">
  import { fade } from "svelte/transition";
  import CardSwiper from "./CardSwiper.svelte";

  let swipe: (direction?: "left" | "right") => void;

  let people = [
    {
      name: "Lucas",
      age: 40,
      description: "Loves to dance in the rain.",
    },
    {
      name: "Benjamin",
      age: 28,
      description: "Eats pizza with a fork.",
    },
    {
      name: "Noah",
      age: 49,
      description: "Talks to plants.",
    },
    {
      name: "Emily",
      age: 45,
      description: "Sleeps with socks on.",
    },
    {
      name: "Ava",
      age: 43,
      description: "Thinks they can speak to animals.",
    },
    {
      name: "Sophia",
      age: 23,
      description: "Obsessed with organizing.",
    },
    {
      name: "Charlotte",
      age: 41,
      description: "Afraid of shadows.",
    },
    {
      name: "Olivia",
      age: 23,
      description: "Collects rare pebbles.",
    },
    {
      name: "Isabella",
      age: 42,
      description: "Always forgets why they walked into a room.",
    },
    {
      name: "Jacob",
      age: 24,
      description: "Can recite movies backwards.",
    },
  ];

  let thresholdPassed = 0;
</script>

// From https://github.com/flo-bit/svelte-swiper-cards
<div
  class="h-[100svh] w-[100svw] p-2 flex items-center justify-center overflow-hidden"
>
  <div class="w-full h-full max-w-xl relative">
    <CardSwiper
      bind:swipe
      cardData={(index) => {
        let i = Math.floor(Math.random() * people.length);
        let j = Math.floor(Math.random() * people.length);
        return {
          title: people[i].name + ", " + people[i].age,
          description: people[j].description,
          image: `/svelte-swiper-cards/profiles/${index % 14}.png`,
        };
      }}
      on:swiped={(e) => {
        console.log(e.detail);
      }}
      bind:thresholdPassed
    />

    <button
      class="absolute bottom-1 left-1 p-3 px-4 bg-white/50 backdrop-blur-sm rounded-full z-10 text-3xl"
      on:click={() => swipe("left")}
    >
      👎
    </button>
    <button
      class="absolute bottom-1 right-1 p-3 px-4 bg-white/50 backdrop-blur-sm rounded-full z-10 text-3xl"
      on:click={() => swipe("right")}
    >
      👍
    </button>
  </div>

  {#if thresholdPassed !== 0}
    <div
      transition:fade={{ duration: 200 }}
      class="absolute w-full h-full inset-0 bg-white/20 backdrop-blur-sm flex items-center justify-center text-9xl pointer-events-none"
    >
      {thresholdPassed > 0 ? "👍" : "👎"}
    </div>
  {/if}
</div>

<a
  href="https://github.com/flo-bit/svelte-swiper-cards/"
  target="_blank"
  class="absolute top-0 right-0 bg-white/40 backdrop-blur-sm rounded-bl-xl p-2"
>
  <svg
    fill="currentColor"
    viewBox="0 0 24 24"
    aria-hidden="true"
    class="w-6 h-6"
  >
    <path
      fill-rule="evenodd"
      d="M12 2C6.477 2 2 6.484 2 12.017c0 4.425 2.865 8.18 6.839 9.504.5.092.682-.217.682-.483 0-.237-.008-.868-.013-1.703-2.782.605-3.369-1.343-3.369-1.343-.454-1.158-1.11-1.466-1.11-1.466-.908-.62.069-.608.069-.608 1.003.07 1.531 1.032 1.531 1.032.892 1.53 2.341 1.088 2.91.832.092-.647.35-1.088.636-1.338-2.22-.253-4.555-1.113-4.555-4.951 0-1.093.39-1.988 1.029-2.688-.103-.253-.446-1.272.098-2.65 0 0 .84-.27 2.75 1.026A9.564 9.564 0 0112 6.844c.85.004 1.705.115 2.504.337 1.909-1.296 2.747-1.027 2.747-1.027.546 1.379.202 2.398.1 2.651.64.7 1.028 1.595 1.028 2.688 0 3.848-2.339 4.695-4.566 4.943.359.309.678.92.678 1.855 0 1.338-.012 2.419-.012 2.747 0 .268.18.58.688.482A10.019 10.019 0 0022 12.017C22 6.484 17.522 2 12 2z"
      clip-rule="evenodd"
    />
  </svg>
</a>
