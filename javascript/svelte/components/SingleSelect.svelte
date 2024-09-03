<script>
  import { afterUpdate } from "svelte";
  import MultiSelect from "svelte-multiselect";

  export let items = [
    {
      id: 2,
      object: {
        name: "Cherry",
        profileImage: {
          photo: "https://your-image-url.jpg",
        },
      },
    },
    {
      id: 3,
      object: {
        name: "Pear",
        profileImage: {
          photo: "https://another-image-url.jpg",
        },
      },
    },
  ];

  export let formInputName = "input_name";
  export let placeholder = "Select options...";
  export let componentID = "searchable-dropdown";
  export let submitButtonText = null;

  let debounceTimer;
  const debouncePeriod = 500; // Debounce period in milliseconds
  let canDispatch = true; // Flag to control event dispatch within debounce period

  let selected = []; // This will hold your selection

  // Function to extract names for display
  // Create a new array that includes a label for each item
  let displayItems = items.map((item) => ({
    id: item.id,
    label: item.object.name,
    photo:
      (item.object.profileImage && item.object.profileImage.thumbnail_url) ||
      null,
  }));

  // Dispatch the event after updates, specifically after hidden inputs have been added
  afterUpdate(() => {
    if (selected.length > 0 && canDispatch) {
      dispatchSelectedEvent(selected);
      canDispatch = false; // Prevent further dispatches
      // Reset canDispatch flag after the debounce period
      clearTimeout(debounceTimer);
      debounceTimer = setTimeout(() => {
        canDispatch = true;
      }, debouncePeriod);
    }
  });

  function dispatchSelectedEvent(selected) {
    const dropdownElement = document.getElementById(componentID);
    if (dropdownElement) {
      const customEvent = new CustomEvent("dropdownSelectionChanged", {
        detail: { selected },
      });
      dropdownElement.dispatchEvent(customEvent);
    }
  }
</script>

<MultiSelect
  bind:selected
  options={displayItems}
  {placeholder}
  outerDivClass="!w-full !input !input-bordered !border !border-gray-300 !rounded-md !p-2.5 !bg-white !text-gray-900 !text-sm md:!text-base focus:!ring-blue-500 focus:!border-blue-500 dark:!bg-gray-700 dark:!border-gray-600 dark:!placeholder-gray-400 dark:!text-white dark:focus:!ring-blue-500 dark:focus:!border-blue-500"
  liSelectedClass="!bg-orange-500 dark:!bg-blue-600 !p-2 !text-white"
  ulOptionsClass="!p-1 !m-1 !bg-white dark:!bg-gray-700 !text-slate-600 dark:!text-white"
  liUserMsgClass="!p-1 !m-1 !bg-white dark:!bg-gray-700 !text-slate-600 dark:!text-white"
  liActiveOptionClass="!bg-slate-200 dark:!bg-blue-700 !rounded-lg"
  maxSelect={1}
  maxSelectMsg={(current, max) => `${current} of ${max} selected`}
>
  <div let:option {option} slot="option" class="flex flex-row items-center">
    {#if option.photo}
      <img
        class="w-10 h-10 rounded-full mr-3 md:mr-5"
        src={option.photo}
        alt="Rounded avatar"
      />
    {/if}
    <span>{option.label}</span>
  </div>
</MultiSelect>

{#if selected.length > 0 && submitButtonText}
  <div class="flex justify-center mt-4 m-10">
    <button
      class="bg-blue-600 hover:bg-blue-500 text-white py-1 px-4 rounded-lg inline-flex items-center"
      aria-label="Finish onboarding"
    >
      <span class="m-2 font-semibold">{submitButtonText}</span>
    </button>
  </div>{/if}
<!-- Hidden inputs for form submission. Adjust to submit the ID of the selected option. -->

{#each selected as option}
  <input type="hidden" name={formInputName} value={option.id} />
{/each}
