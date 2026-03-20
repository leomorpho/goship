<script>
  import PhotoUploader from "../javascript/svelte/components/PhotoUploader.svelte";

  export let postURL = "localhost:8000/uploadImage";
  export let uploadEnabled = true;
  export let stylingClassesUploadEnabled = "";
  export let stylingClassesUploadDisabled = "";
  export let buttonText = "Upload image";
  export let isProfilePhotoCornerUploader = false;
</script>

<PhotoUploader
  {postURL}
  {uploadEnabled}
  {stylingClassesUploadEnabled}
  {stylingClassesUploadDisabled}
  {buttonText}
  {isProfilePhotoCornerUploader}
/>
