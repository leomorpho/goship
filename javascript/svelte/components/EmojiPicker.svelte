<script>
  import { afterUpdate, onDestroy, onMount } from "svelte";

  export let id;
  export let eventName = "EmojiPicker";
  export let formUnifiedName = "emoji_unified";
  export let formShorthandName = "emoji_shorthand";
  export let formAddName = "emoji_add";

  let debounceTimer;
  const debouncePeriod = 500; // Debounce period in milliseconds
  let canDispatch = true; // Flag to control event dispatch within debounce period

  let emojiPickerContainer;
  let pickerInstance;
  let pickerContainer; // This variable will hold our intermediary container element
  let popperInstance = null;

  // TODO: this should not be a list
  let selected = []; // This will hold your selection

  let emojiPickerVisible = false;

  async function loadEmojiMark() {
    if (!window.EmojiMart) {
      // Dynamically import EmojiMart
      await import(
        "https://cdn.jsdelivr.net/npm/emoji-mart@latest/dist/browser.js"
      );
    }
    if (!window.Popper) {
      await import("https://unpkg.com/@popperjs/core@2");
    }
  }

  // Function to check current theme and return 'dark' or 'light'
  function getCurrentTheme() {
    return document.documentElement.getAttribute("data-theme") === "darkmode"
      ? "dark"
      : "light";
  }

  function handleEmojiSelect(emoji) {
    selected = [emoji];
    toggleEmojiPicker();
  }

  // Dispatch the event after updates, specifically after hidden inputs have been added
  afterUpdate(() => {
    if (selected.length > 0 && canDispatch) {
      dispatchEmojiEvent(selected);
      selected = []; // Clear the selection after dispatching
      canDispatch = false; // Prevent further dispatches
      // Reset canDispatch flag after the debounce period
      clearTimeout(debounceTimer);
      debounceTimer = setTimeout(() => {
        canDispatch = true;
      }, debouncePeriod);
    }
  });

  function dispatchEmojiEvent(selected) {
    const dropdownElement = document.getElementById(id);
    if (dropdownElement) {
      const customEvent = new CustomEvent(eventName, {
        detail: { selected },
      });
      dropdownElement.dispatchEvent(customEvent);
    }
  }

  function toggleEmojiPicker(event) {
    // Prevent any form submission or default action
    if (event) {
      event.preventDefault();
    }

    // Prevent immediate hiding after showing
    setTimeout(() => {
      emojiPickerVisible = !emojiPickerVisible;
      if (emojiPickerVisible) {
        showPicker();
      } else {
        hidePicker();
      }
    }, 50); // Short delay to bypass immediate click outside detection
  }

  function showPicker() {
    emojiPickerContainer = document.getElementById(id);

    // Destroy existing picker instance if it exists
    if (pickerInstance && typeof pickerInstance.destroy === "function") {
      pickerInstance.destroy();
    }
    // Remove previous picker element from DOM and clear reference
    if (pickerContainer && pickerContainer.firstChild) {
      pickerContainer.removeChild(pickerContainer.firstChild);
    }

    const currentTheme = getCurrentTheme();
    pickerInstance = new EmojiMart.Picker({
      theme: currentTheme,
      autoFocus: true,
      previewPosition: "none",
      categories: [
        "frequent",
        "people",
        "nature",
        "objects",
        "foods",
        "places",
        "activity",
        "flags",
        "symbols",
      ],
      onEmojiSelect: (emoji) => {
        handleEmojiSelect(emoji);
      },
    });

    if (!pickerContainer) {
      pickerContainer = document.createElement("div");
      pickerContainer.classList.add("picker-container");
      document.body.appendChild(pickerContainer); // Append to body to manage positioning globally
    }

    pickerContainer.appendChild(pickerInstance);

    if (!popperInstance) {
      popperInstance = Popper.createPopper(
        emojiPickerContainer,
        pickerContainer,
        {
          placement: "bottom-start",
          strategy: "fixed", // Ensures smart positioning
          modifiers: [
            {
              name: "flip",
              options: {
                fallbackPlacements: ["top-start", "bottom-end", "top-end"], // Adjust based on preference
              },
            },
            {
              name: "preventOverflow",
              options: {
                boundary: "viewport",
                padding: 4,
              },
            },
          ],
        }
      );
    }

    // Schedule the Popper update to run after the picker becomes visible and possibly changes size
    setTimeout(() => {
      if (popperInstance) {
        popperInstance.update();
      }
    }, 150); // You may need to adjust this timeout based on how quickly your picker contents stabilize

    // TODO: the below is hacky but it's to cover my bases on slow connections
    setTimeout(() => {
      if (popperInstance) {
        popperInstance.update();
      }
    }, 300);
    setTimeout(() => {
      if (popperInstance) {
        popperInstance.update();
      }
    }, 1500);
    setTimeout(() => {
      if (popperInstance) {
        popperInstance.update();
      }
    }, 3000);
  }

  function hidePicker() {
    if (pickerContainer && pickerContainer.parentNode) {
      pickerContainer.parentNode.removeChild(pickerContainer);
      pickerContainer = null;
    }
    if (popperInstance) {
      popperInstance.destroy();
      popperInstance = null;
    }
  }

  function handleClickOutside(event) {
    if (
      emojiPickerVisible &&
      pickerContainer &&
      !pickerContainer.contains(event.target) &&
      !event.target.closest(".emoji-mart")
    ) {
      hidePicker();
    }
  }

  onMount(() => {
    loadEmojiMark();
    emojiPickerContainer = document.getElementById(id);
    document.addEventListener("click", handleClickOutside);
    emojiPickerContainer = document.getElementById(id);
  });

  onDestroy(() => {
    document.removeEventListener("click", handleClickOutside);

    if (popperInstance) {
      popperInstance.destroy();
    }
    pickerInstance = null;
    emojiPickerContainer = null;
  });
</script>

<button on:click|stopPropagation={toggleEmojiPicker}>
  <div
    class="flex bg-slate-100 dark:bg-slate-700 hover:bg-slate-300 dark:hover:bg-slate-500 first-line:text-slate-900 dark:text-slate-200 rounded-2xl p-1.5 px-2 items-center text-bold"
  >
    <svg
      xmlns="http://www.w3.org/2000/svg"
      fill="none"
      viewBox="0 0 24 24"
      stroke-width="1.7"
      stroke="currentColor"
      class="w-5 h-5"
    >
      <path
        stroke-linecap="round"
        stroke-linejoin="round"
        d="M15.182 15.182a4.5 4.5 0 0 1-6.364 0M21 12a9 9 0 1 1-18 0 9 9 0 0 1 18 0ZM9.75 9.75c0 .414-.168.75-.375.75S9 10.164 9 9.75 9.168 9 9.375 9s.375.336.375.75Zm-.375 0h.008v.015h-.008V9.75Zm5.625 0c0 .414-.168.75-.375.75s-.375-.336-.375-.75.168-.75.375-.75.375.336.375.75Zm-.375 0h.008v.015h-.008V9.75Z"
      />
    </svg>
  </div>
</button>

{#if emojiPickerVisible}
  <div class="emoji-picker-container" bind:this={emojiPickerContainer}>
    <!-- The Picker will be appended here -->
  </div>
{/if}

<!-- Hidden inputs for form submission. Adjust to submit the ID of the selected option. -->
{#each selected as optionName}
  {#if optionName}
    <input type="hidden" name={formUnifiedName} value={optionName.unified} />
    <input
      type="hidden"
      name={formShorthandName}
      value={optionName.shortcodes}
    />
    <input type="hidden" name={formAddName} value={true} />
  {/if}
{/each}
