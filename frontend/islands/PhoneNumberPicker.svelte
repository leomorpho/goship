<script>
  import PhoneNumberPicker from "../javascript/svelte/components/PhoneNumberPicker.svelte";

  export let value = null;
  export let country = null;
  export let valid;
  export let detailedValue = null;
  export let options;
  export let disabled = false;
  export let readonly = false;
  export let formInputNameE164 = "phone_number_e164";
  export let formInputNameCountryCode = "country_code";
  export let saveEventName = "savePhoneNumber";
  export let componentID = "phone-number-picker";
</script>

<PhoneNumberPicker
  bind:value
  bind:country
  bind:valid
  bind:detailedValue
  {options}
  {disabled}
  {readonly}
  {formInputNameE164}
  {formInputNameCountryCode}
  {saveEventName}
  {componentID}
/>
