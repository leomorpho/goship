<script>
  import { Crop, Local, Uppload, en } from "uppload";
  import "uppload/dist/themes/light.css";
  import "uppload/dist/uppload.css";

  export let postURL = "localhost:8000/uploadImage";
  export let uploadEnabled = true; // Prop to control button visibility
  export let stylingClassesUploadEnabled = "";
  export let stylingClassesUploadDisabled = "";
  export let buttonText = "Upload image";
  export let isProfilePhotoCornerUploader = false;
  let uploader;

  uploader = new Uppload({
    lang: en, // Use English language
    uploader: (file, updateProgress) =>
      new Promise((resolve, reject) => {
        const formData = new FormData();
        formData.append("file", file);

        const xhr = new XMLHttpRequest();
        xhr.open("POST", postURL, true);

        xhr.upload.onprogress = (event) => {
          if (event.lengthComputable) {
            const progress = event.loaded / event.total;
            updateProgress(progress); // Update upload progress
          }
        };

        xhr.onload = () => {
          if (xhr.status >= 200 && xhr.status < 300) {
            // Reload the current page
            window.location.reload();
            resolve("Upload successful");
          } else {
            reject("Upload failed"); // Reject the promise
          }
        };

        xhr.onerror = () => reject("Upload error");
        xhr.send(formData); // Send the request with form data
      }),
    maxSize: [1500, 1500],
    compression: 0.8,
    compressionToMime: "image/jpeg",
  });

  // Use the Local service for uploading images from local device
  uploader.use([new Local()]);

  // Enable square cropping
  uploader.use([
    new Crop({
      aspectRatio: 1, // Square crop
    }),
  ]);

  // Open Uppload on button click
  const openUploader = () => uploader.open();

  // Optionally, listen to other Uppload events as needed
  uploader.on("upload", (url) => {
    console.log("File uploaded!", url);
    // Handle the uploaded image URL (e.g., display it or send it to your server)
  });
</script>

<button
  on:click={openUploader}
  class={uploadEnabled
    ? stylingClassesUploadEnabled
    : stylingClassesUploadDisabled}
  disabled={!uploadEnabled}
>
  {#if isProfilePhotoCornerUploader}
    <div
      class="absolute inline-flex items-center justify-center w-6 h-6
				      text-xs font-bold text-white bg-gray-800 border-2 rounded-full
				      bottom-4 left-3/4 transform -translate-x-1/2 translate-y-1/2"
    >
      <svg
        xmlns="http://www.w3.org/2000/svg"
        fill="none"
        viewBox="0 0 24 24"
        stroke-width="1.5"
        stroke="currentColor"
        class="w-3 h-3"
      >
        <path
          stroke-linecap="round"
          stroke-linejoin="round"
          d="M12 4.5v15m7.5-7.5h-15"
        ></path>
      </svg>
    </div>
  {:else}
    <div class="flex flex-row items-center">
      <svg
        xmlns="http://www.w3.org/2000/svg"
        viewBox="0 0 20 20"
        fill="currentColor"
        class="w-5 h-5 mr-1 md:mr-2"
      >
        <path
          fill-rule="evenodd"
          d="M1 5.25A2.25 2.25 0 0 1 3.25 3h13.5A2.25 2.25 0 0 1 19 5.25v9.5A2.25 2.25 0 0 1 16.75 17H3.25A2.25 2.25 0 0 1 1 14.75v-9.5Zm1.5 5.81v3.69c0 .414.336.75.75.75h13.5a.75.75 0 0 0 .75-.75v-2.69l-2.22-2.219a.75.75 0 0 0-1.06 0l-1.91 1.909.47.47a.75.75 0 1 1-1.06 1.06L6.53 8.091a.75.75 0 0 0-1.06 0l-2.97 2.97ZM12 7a1 1 0 1 1-2 0 1 1 0 0 1 2 0Z"
          clip-rule="evenodd"
        />
      </svg>

      {buttonText}
    </div>
  {/if}
</button>

<style>
  @import "uppload/dist/uppload.css";
  @import "uppload/dist/themes/dark.css";
</style>
