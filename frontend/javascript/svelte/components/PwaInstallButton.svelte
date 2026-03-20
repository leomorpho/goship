<script>
  import "@khmyznikov/pwa-install";
  import { onMount } from "svelte";
  import { writable } from "svelte/store";

  export let position = "navbar";

  let pwaInstall;
  let id = `pwa-install-${generateUUID()}`;
  let installable = writable(false);

  // Function to generate a UUID
  function generateUUID() {
    return crypto.randomUUID();
  }

  onMount(() => {
    pwaInstall = document.getElementById(id);
    if (pwaInstall.isUnderStandaloneMode) {
      installable.set(false); // Set to false if it's already in standalone mode
    } else {
      console.log("PWA install is available.");
      installable.set(true); // Set to true if install is available
    }
  });

  function isIosButNotSafari() {
    var ua = window.navigator.userAgent;
    var iOS = !!ua.match(/iPad/i) || !!ua.match(/iPhone/i);
    var webkit = !!ua.match(/WebKit/i);
    var isSafari = !!ua.match(/Safari/i) && !ua.match(/CriOS/i);

    return iOS && webkit && !isSafari;
  }

  const forceStyle = (style) => {
    switch (style) {
      case "apple-mobile":
        pwaInstall.isAppleDesktopPlatform = false;
        pwaInstall.isAppleMobilePlatform = true;
        break;
      case "apple-desktop":
        pwaInstall.isAppleMobilePlatform = false;
        pwaInstall.isAppleDesktopPlatform = true;
        break;
      case "chrome":
        pwaInstall.isAppleMobilePlatform = false;
        pwaInstall.isAppleDesktopPlatform = false;
        break;
    }
    pwaInstall.hideDialog();
  };

  function detectPlatform() {
    const userAgent = window.navigator.userAgent;

    var ua = window.navigator.userAgent;
    var iOS = !!ua.match(/iPad/i) || !!ua.match(/iPhone/i);
    var webkit = !!ua.match(/WebKit/i);
    var iOSSafari = iOS && webkit && !ua.match(/CriOS/i);

    if (iOSSafari) {
      return "mobile-safari";
    } else if (/Android/.test(userAgent)) {
      return "android";
    } else if (webkit) {
      return "desktop-safari";
    } else {
      return "not-safari";
    }
  }

  function showInstallDialog() {
    if (isIosButNotSafari()) {
      Swal.fire({
        icon: "error",
        title: "Oops...",
        text: "Unfortunately, this app can only be installed using the Safari browser on iOS due to platform restrictions 😥.",
      });
      return;
    }

    if (pwaInstall) {
      if (detectPlatform() == "mobile-safari") {
        console.log("opening safari pwa install dialogue window");
        forceStyle("apple-mobile");
      } else if (detectPlatform() == "android") {
        console.log("opening desktop safari pwa install dialogue window");
        forceStyle("chrome");
      } else if (detectPlatform() == "desktop-safari") {
        console.log("opening chrome pwa install dialogue window");
        forceStyle("apple-desktop");
      } else {
        console.log("opening chrome pwa install dialogue window");
        forceStyle("chrome");
      }
      pwaInstall.showDialog(true);
    } else {
      console.error(
        "PWA Install element is not ready or showDialog is not a function."
      );
    }
  }

  let buttonClasses =
    "bg-gradient-to-r from-purple-500 to-purple-900 text-white font-medium rounded-full flex justify-center items-center m-1";
  if (position === "navbar") {
    buttonClasses += " w-32 p-1";
  } else if (position === "landing-page") {
    buttonClasses += " w-48 sm:w-64 p-2 sm:p-3";
  } else if (position === "mobile") {
    buttonClasses += " w-full p-2";
  }
</script>

{#if $installable}
  <button on:click={showInstallDialog} class={buttonClasses}>
    <svg
      xmlns="http://www.w3.org/2000/svg"
      viewBox="0 0 20 20"
      fill="currentColor"
      class="w-5 h-5 mr-1 sm:mr-2"
    >
      <path
        fill-rule="evenodd"
        d="M5.5 17a4.5 4.5 0 0 1-1.44-8.765 4.5 4.5 0 0 1 8.302-3.046 3.5 3.5 0 0 1 4.504 4.272A4 4 0 0 1 15 17H5.5Zm5.25-9.25a.75.75 0 0 0-1.5 0v4.59l-1.95-2.1a.75.75 0 1 0-1.1 1.02l3.25 3.5a.75.75 0 0 0 1.1 0l3.25-3.5a.75.75 0 1 0-1.1-1.02l-1.95 2.1V7.75Z"
        clip-rule="evenodd"
      />
    </svg>

    <div>Click to Install</div>
  </button>
{/if}

<pwa-install
  class="z-50"
  name="Goship"
  description="Your Quick Productization Tool."
  manifest-url="/files/manifest.json"
  icon="https://chatbond-static.s3.us-west-002.backblazeb2.com/cherie/pwa/manifest-icon-192.maskable.png"
  {id}
></pwa-install>

<style>
  dialog::backdrop {
    background: rgba(0, 0, 0, 0.5);
  }
</style>
