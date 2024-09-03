<script>
  export let sliderName = "Name";
  export let startScaleName = "Low";
  export let endScaleName = "High";
  export let value = "50";
</script>

<div class="bg-slate-200 dark:bg-slate-600 rounded-lg m-3 sm:m-5 p-2 sm:p-4">
  <div class="flex justify-between items-center">
    <span>{startScaleName}</span>
    <label
      for="minmax-range"
      class="block mb-2 text-sm font-medium text-gray-900 dark:text-white"
      >{sliderName}</label
    >
    <span>{endScaleName}</span>
  </div>
  <input
    id="minmax-range"
    type="range"
    min="0"
    max="100"
    {value}
    step="1"
    class="w-full h-2 bg-gray-400 rounded-lg appearance-none cursor-pointer dark:bg-gray-700"
  />
</div>
