<script>
  import { onMount } from "svelte";
  import MultiSelect from "svelte-multiselect";

  export let items = [`Svelte`, `React`, `Vue`, `Angular`, `...`];
  export let selected = [];
  export let placeholder = "Select options...";
  export let formInputName = "input_name";
  export let postURL = "/multiselect-post-request";
  export let csrfToken = "";

  let previousSelected = [];

  // Ensure 'selected' is always an array
  $: selected = Array.isArray(selected) ? selected : [];

  onMount(() => {
    previousSelected = [...selected];
  });

  // Reactively send POST request when 'selected' changes
  $: if (selected.length !== previousSelected.length) {
    submitForm();
    previousSelected = [...selected];
  }

  async function submitForm() {
    const formData = new FormData();

    // Assuming `selected` contains the selected interested genders
    selected.forEach((gender) => {
      formData.append("interested_genders", gender);
    });

    // Append other form data if needed
    // formData.append('otherField', otherFieldValue);

    try {
      const response = await fetch(postURL, {
        method: "POST",
        headers: {
          "X-CSRF-Token": csrfToken,
        },
        body: formData,
        credentials: "include",
      });

      if (!response.ok) {
        throw new Error(`HTTP error! status: ${response.status}`);
      }

      // Handle successful form submission
      const data = await response.json();
      console.log("Form submitted successfully:", data);
    } catch (error) {
      console.error("Failed to submit form:", error);
    }
  }
</script>

<MultiSelect
  bind:selected
  options={items}
  {placeholder}
  outerDivClass="!bg-gray-50 !w-full !input !input-bordered !border-gray-300 
  !rounded-lg !p-2.5 !text-gray-900 !text-sm
  focus:!ring-blue-500 focus:!border-blue-500
  dark:!bg-gray-700 dark:!border-gray-600 dark:!placeholder-gray-400 dark:!text-white 
  dark:focus:!ring-blue-500 dark:focus:!border-blue-500
  "
  liSelectedClass="!bg-orange-500 dark:!bg-blue-600 !p-2 !text-white"
  ulOptionsClass="!p-2 !m-1 !bg-gray-50 !text-gray-900 !text-sm dark:!bg-gray-700 dark:!border-gray-600 dark:!text-white "
  liOptionClass="!p-2 !m-1 !bg-gray-50 !text-gray-900 !text-sm dark:!bg-gray-700 dark:!border-gray-600 dark:!text-white hover:!bg-gray-200 hover:dark:!bg-gray-500 rounded"
  liUserMsgClass="!p-2 !m-1 !bg-gray-50 !text-gray-900 !text-sm dark:!bg-gray-700 dark:!border-gray-600 dark:!text-white "
  liActiveOptionClass="!bg-slate-200 dark:!bg-blue-300"
  allowUserOptions={true}
  createOptionMsg="Hit enter to create"
/>

<!-- Hidden inputs for form submission -->
{#each selected as option}
  <input type="hidden" name={formInputName} value={option} />
{/each}
