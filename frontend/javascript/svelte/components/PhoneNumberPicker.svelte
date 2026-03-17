<script lang="ts">
  import { afterUpdate } from "svelte";
  import { TelInput, normalizedCountries } from "svelte-tel-input";
  import type {
    CountryCode,
    DetailedValue,
    E164Number,
    TelInputOptions,
  } from "svelte-tel-input/types";

  // E164 formatted value, usually you should store and use this.
  export let value: E164Number | null = null;

  // Selected country
  export let country: CountryCode | null = null;

  // Validity
  export let valid: boolean;

  // Phone number details
  export let detailedValue: DetailedValue | null = null;

  export let options: TelInputOptions;

  export let disabled: boolean = false;
  export let readonly: boolean = false;

  export let formInputNameE164: string = "phone_number_e164";
  export let formInputNameCountryCode: string = "country_code";

  export let saveEventName: string = "savePhoneNumber";
  export let componentID = "phone-number-picker";

  normalizedCountries.sort();

  let debounceTimer: any;
  const debouncePeriod = 200; // Debounce period in milliseconds
  let canDispatch = true; // Flag to control event dispatch within debounce period

  // Dispatch the event after updates, specifically after hidden inputs have been added
  afterUpdate(() => {
    if (canSubmit) {
      dispatchSelectedEvent(detailedValue);
      canDispatch = false; // Prevent further dispatches
      // Reset canDispatch flag after the debounce period
      clearTimeout(debounceTimer);
      debounceTimer = setTimeout(() => {
        canDispatch = true;
      }, debouncePeriod);
    }
  });

  function dispatchSelectedEvent(detailedValue: any) {
    const dropdownElement = document.getElementById(componentID);
    if (dropdownElement) {
      const customEvent = new CustomEvent(saveEventName, {
        detail: { detailedValue },
      });
      dropdownElement.dispatchEvent(customEvent);
    }
  }
  $: canSubmit = value && detailedValue && valid;

  // $: console.log("detailedValue:", detailedValue);
</script>

<div class="flex w-full">
  <select
    class="form-select appearance-none w-24 sm:w-40
  bg-gray-50 border border-gray-300 text-gray-900 text-xs sm:text-sm focus:ring-blue-500 focus:border-blue-500
    block p-2.5 dark:bg-gray-700 dark:border-gray-600 dark:placeholder-gray-400
  dark:text-white dark:focus:ring-blue-500 dark:focus:border-blue-500
    bg-clip-padding bg-no-repeat cursor-pointer
    rounded-l-lg
    m-0 focus:outline-none"
    aria-label="Default select example"
    name="Country"
    bind:value={country}
  >
    <option value={null} hidden={country !== null}>Country</option>
    {#each normalizedCountries as currentCountry (currentCountry.id)}
      <option
        value={currentCountry.iso2}
        selected={currentCountry.iso2 === country}
        aria-selected={currentCountry.iso2 === country}
      >
        {currentCountry.iso2} (+{currentCountry.dialCode})
      </option>
    {/each}
  </select>

  <TelInput
    required={true}
    {options}
    bind:country
    bind:valid
    bind:value
    bind:detailedValue
    bind:disabled
    bind:readonly
    class="bg-gray-50  text-gray-900 text-xs sm:text-sm 
    block p-2.5 dark:bg-gray-700  dark:placeholder-gray-400
  dark:text-white 
     rounded-r-lg w-full {valid
      ? 'border border-gray-300 border-l-gray-100 dark:border-l-gray-700 dark:border-gray-600'
      : 'border-2 border-red-600'}"
  />
</div>

<!-- Only render the hidden input when the phone number is valid -->
{#if canSubmit && detailedValue}
  <input type="hidden" name={formInputNameE164} value={detailedValue.e164} />
  <input
    type="hidden"
    name={formInputNameCountryCode}
    value={detailedValue.countryCode}
  />
{/if}
