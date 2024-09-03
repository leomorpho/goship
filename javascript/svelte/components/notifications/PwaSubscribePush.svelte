<script>
  import { onMount } from "svelte";
  import { toast } from "wc-toast";
  import PermissionButton from "./PermissionButton.svelte";
  import LoadingSpinner from "./icons/LoadingSpinner.svelte";
  import PushPermissionDisabledIcon from "./icons/PushDisabledIcon.svelte";
  import PushPermissionEnabledIcon from "./icons/PushEnabledIcon.svelte";

  export let vapidPublicKey = "";
  export let subscribedEndpoints = [];
  export let addSubscriptionEndpoint = "/subscribe-to-push-notifs";
  export let deleteSubscriptionEndpoint = "/unsubscribe-from-push-notifs";

  export let permissionKey = "";
  export let permissionGranted = false;
  export let notificationTypeQueryParamKey = "";

  let isSubscribedForPermission = false;
  let isAppInstalled = false; // State to track if app is in standalone mode
  let isAppInstallable = true;
  let currentSubscription = null; // PushSubscription | null
  let loading = false;
  let iOSPushCapability = false;

  function isBrowserOnIOS() {
    var ua = window.navigator.userAgent;
    var webkit = !!ua.match(/WebKit/i);
    var iOS = !!ua.match(/iPad/i) || !!ua.match(/iPhone/i);

    if (webkit && iOS) {
      return true;
    }
    return false;
  }

  async function loadDynamicDependencies() {
    if (!window.Swal) {
      await import("https://cdn.jsdelivr.net/npm/sweetalert2@10");
    }
  }

  onMount(async () => {
    await loadDynamicDependencies();

    // Check for standalone mode in Safari on iOS
    let isStandalone = window.navigator.standalone;
    isAppInstalled =
      isStandalone || window.matchMedia("(display-mode: standalone)").matches;
    if (!isAppInstalled) {
      isAppInstallable = !(isBrowserOnIOS() && !isStandalone);
    }

    if ("serviceWorker" in navigator) {
      const registration = await navigator.serviceWorker.ready;
      if (registration.pushManager) {
        currentSubscription = await registration.pushManager.getSubscription();
        isSubscribedForPermission =
          permissionGranted && checkIfDeviceIsSubscribed();
      }
    }
  });

  function checkIfDeviceIsSubscribed() {
    if (!currentSubscription || !subscribedEndpoints) return false;
    return subscribedEndpoints.includes(currentSubscription.endpoint);
  }

  function isIosButNotSafari() {
    var ua = window.navigator.userAgent;
    var iOS = !!ua.match(/iPad/i) || !!ua.match(/iPhone/i);
    var webkit = !!ua.match(/WebKit/i);
    var isSafari = !!ua.match(/Safari/i) && !ua.match(/CriOS/i);

    return iOS && webkit && !isSafari;
  }

  async function subscribeUser() {
    loading = true;

    if (isIosButNotSafari()) {
      Swal.fire({
        icon: "error",
        title: "Oops...",
        text: "Please install the app from the Apple App Store to get push notifications.",
      });
      loading = false;
      return;
    }

    if (!isAppInstallable) {
      loading = false;
      console.log("platform does not support push notifications");
      return;
    }

    navigator.serviceWorker.register("/service-worker.js");

    try {
      const registration = await navigator.serviceWorker.ready;

      if (currentSubscription) {
        await currentSubscription.unsubscribe();
      }

      const subscription = await registration.pushManager.subscribe({
        userVisibleOnly: true,
        applicationServerKey: urlB64ToUint8Array(vapidPublicKey),
      });

      try {
        await sendSubscriptionToBackend(subscription);
        currentSubscription = subscription;
        isSubscribedForPermission = true;
        toast.success("Push notification turned on");
      } catch (backendError) {
        toast.error(
          `Failed to subscribe: ${backendError.message || "Unknown error"}`
        );
        // If sending to the backend fails, unsubscribe to clean up and revert UI
        console.error("Failed to send subscription to backend: ", backendError);
        await subscription.unsubscribe();
        throw backendError;
      }
    } catch (error) {
      toast.error(`Failed to subscribe: ${error.message || "Unknown error"}`);
      console.error("Failed to subscribe the user: ", error);
      setTimeout(() => {
        isSubscribedForPermission = false;
      }, 250);
    } finally {
      loading = false;
    }
  }

  async function unsubscribeUser() {
    loading = true;
    if (iOSPushCapability && isSubscribedForPermission) {
      await unsubscribeIosNativePushPermission(
        deleteIosNativeSubscriptionEndpoint
      );
      loading = false;
    } else if (currentSubscription) {
      try {
        try {
          await removeSubscriptionFromBackend(currentSubscription); // Notify backend
          currentSubscription = null;
          isSubscribedForPermission = false; // Update UI and state only if backend update succeeds
          toast.success("Push notification turned off");
        } catch (backendError) {
          toast.error(
            `Failed to unsubscribe: ${backendError.message || "Unknown error"}`
          );
          isSubscribedForPermission = true;
        }
      } catch (unsubscribeError) {
        console.error(
          "Failed to unsubscribe the user locally: ",
          unsubscribeError
        );
        toast.error(
          `Failed to unsubscribe: ${unsubscribeError.message || "Unknown error"}`
        );
      }
    }
    loading = false;
  }

  async function sendSubscriptionToBackend(subscription) {
    let url =
      addSubscriptionEndpoint +
      `&${notificationTypeQueryParamKey}=` +
      permissionKey;
    const response = await fetch(url, {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
      },
      body: JSON.stringify(subscription),
    });
    if (!response.ok) {
      throw new Error(
        `Failed to add subscription: ${response.status} ${response.statusText}`
      );
    }
  }

  async function removeSubscriptionFromBackend(subscription) {
    let url =
      deleteSubscriptionEndpoint +
      `&${notificationTypeQueryParamKey}=` +
      permissionKey;
    const response = await fetch(url, {
      method: "DELETE",
      headers: {
        "Content-Type": "application/json",
      },
      body: JSON.stringify({ endpoint: subscription.endpoint }),
    });
    if (!response.ok) {
      throw new Error(
        `Failed to remove subscription: ${response.status} ${response.statusText}`
      );
    }
  }

  function urlB64ToUint8Array(base64String) {
    const padding = "=".repeat((4 - (base64String.length % 4)) % 4);
    const base64 = (base64String + padding)
      .replace(/\-/g, "+")
      .replace(/_/g, "/");
    const rawData = atob(base64);
    const outputArray = new Uint8Array(rawData.length);
    for (let i = 0; i < rawData.length; ++i) {
      outputArray[i] = rawData.charCodeAt(i);
    }
    return outputArray;
  }
</script>

<wc-toast></wc-toast>

{#if loading}
  <PermissionButton {permissionGranted}>
    <LoadingSpinner slot="icon" />
  </PermissionButton>
{:else if isSubscribedForPermission}
  <PermissionButton
    on:click={unsubscribeUser}
    permissionGranted={isSubscribedForPermission}
  >
    <PushPermissionEnabledIcon slot="icon" />
  </PermissionButton>
{:else}
  <PermissionButton
    on:click={subscribeUser}
    permissionGranted={isSubscribedForPermission}
  >
    <PushPermissionDisabledIcon slot="icon" />
  </PermissionButton>
{/if}
