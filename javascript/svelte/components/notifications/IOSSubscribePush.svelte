<script>
  import { onMount } from "svelte";
  import { toast } from "wc-toast";
  import PermissionButton from "./PermissionButton.svelte";
  import LoadingSpinner from "./icons/LoadingSpinner.svelte";
  import PushPermissionDisabledIcon from "./icons/PushDisabledIcon.svelte";
  import PushPermissionEnabledIcon from "./icons/PushEnabledIcon.svelte";

  export let addSubscriptionEndpoint = "/subscribe-to-native-ios-push-notifs";
  export let deleteSubscriptionEndpoint =
    "/unsubscribe-from-native-ios-push-notifs";

  export let permissionKey = "";
  export let permissionGranted = false;
  export let notificationTypeQueryParamKey = "";

  let isSubscribedForPermission = false;
  let loading = false;
  let iOSPushCapability = false;

  let fcmToken = "";
  // requestedPermission tracks when a token is requested for UI or whether user actually requested perms.
  let requestedPermission = false;
  let tokenProcessed = false;
  let token = "";
  let lastToastTime = 0; // Track the last time a toast was displayed

  onMount(async () => {
    // console.log("window.webkit", window.webkit);
    // console.log("window.webkit.messageHandlers", window.webkit.messageHandlers);
    // console.log(
    //   `window.webkit.messageHandlers["push-permission-request"]`,
    //   window.webkit.messageHandlers["push-permission-request"]
    // );
    // console.log(
    //   `window.webkit.messageHandlers["push-permission-state"]`,
    //   window.webkit.messageHandlers["push-permission-state"]
    // );
    if (
      window.webkit &&
      window.webkit.messageHandlers &&
      window.webkit.messageHandlers["push-permission-request"] &&
      window.webkit.messageHandlers["push-permission-state"]
    ) {
      iOSPushCapability = true;

      setupPWAAPPBuilderHandlers();

      // Issue request that will set the current permission state in the UI
      getIosNativePushPermissionState();
    }
    console.log("iOSPushCapability", iOSPushCapability);
  });

  function debounce(func, wait) {
    let timeout;
    return function (...args) {
      clearTimeout(timeout);
      timeout = setTimeout(() => {
        func.apply(this, args);
      }, wait);
    };
  }

  const debouncedRequestIosNativePushPermission = debounce(
    requestIosNativePushPermission,
    500
  );

  async function subscribeUser() {
    loading = true;
    requestedPermission = true;

    if (!token) {
      debouncedRequestIosNativePushPermission();
    } else {
      subscribeUserWithExistingToken();
    }
  }

  async function unsubscribeUser() {
    loading = true;
    if (isSubscribedForPermission) {
      try {
        await removeSubscriptionFromBackend();
        isSubscribedForPermission = false; // Update UI and state only if backend update succeeds
        toast.success("Push notification turned off");
        tokenProcessed = false;
      } catch (unsubscribeError) {
        console.error("Failed to unsubscribe the user", unsubscribeError);
        toast.error(
          `Failed to unsubscribe: ${unsubscribeError.message || "Unknown error"}`
        );
      }
    }
    loading = false;
  }

  // PWABuilder IOS stuff
  function setupPWAAPPBuilderHandlers() {
    // Listen to custom events from PWAAppBuilder
    window.addEventListener("push-permission-request", (event) =>
      handleIosNativePushPermissionRequest(event)
    );
    window.addEventListener("push-permission-state", (event) =>
      handleIosNativePushPermissionState(event)
    );
    window.addEventListener("push-notification", (event) =>
      handleIosNativePushNotification(event)
    );
    window.addEventListener("push-token", (event) =>
      handleIosNativePushToken(event)
    );
  }

  /*
  REQUESTS TO IOS DEVICE
  */
  // Note that PWABuilder provides a messaging interface, so that we can post a message to
  // the iOS swift layer, and later receive an event from it with the results, as seen in the
  // below request functions.
  function requestIosNativePushPermission() {
    if (iOSPushCapability) {
      window.webkit.messageHandlers["push-permission-request"].postMessage(
        "push-permission-request"
      );
    }
  }

  function getIosNativePushPermissionState() {
    if (iOSPushCapability) {
      window.webkit.messageHandlers["push-permission-state"].postMessage(
        "push-permission-state"
      );
    }
  }

  function requestIosNativePushToken() {
    if (iOSPushCapability) {
      window.webkit.messageHandlers["push-token"].postMessage("push-token");
    }
  }

  /*
  HANDLERS FOR IOS DEVICE RESPONSES FROM REQUESTS
  */
  // PWAAppBuilder handlers
  function handleIosNativePushPermissionRequest(event) {
    if (event.detail) {
      console.log(event.detail);
      switch (event.detail) {
        case "granted":
          requestIosNativePushToken();
          // Delay setting isSubscribedForPermission to true until we've saved the token to the backend.
          break;
        default:
          isSubscribedForPermission = false;
          break;
      }
    }
  }

  function handleIosNativePushPermissionState(event) {
    if (event.detail) {
      console.log(event.detail);
      switch (event.detail) {
        case "notDetermined":
          break;
        case "denied":
          isSubscribedForPermission = false;
          break;
        case "authorized":
          requestIosNativePushToken();
          isSubscribedForPermission = true;
        case "ephemeral":
        case "provisional":
          break;
        default:
          break;
      }
    }
  }

  function handleIosNativePushNotification(event) {
    if (event.detail) {
      console.log("Push notification received:", JSON.stringify(event.detail));
      // TODO: if the user is in app, I don't see why we would want to show anything, except perhaps a banner with the news?
      // toast.success("Push notification received");
    }
  }

  async function handleIosNativePushToken(event) {
    if (!requestedPermission) {
      return;
    }
    token = event.detail;
    await subscribeUserWithExistingToken();
  }

  const subscribeUserWithExistingToken = debounce(async () => {
    if (token && !tokenProcessed) {
      console.log("Push token received:", JSON.stringify(token));
      try {
        fcmToken = JSON.stringify(token);
        console.log("event.detail", token);
        console.log("fcmToken", fcmToken);

        if (requestedPermission) {
          await sendSubscriptionToBackend(token);
          isSubscribedForPermission = true;
          permissionGranted = true;

          // Reset requestedPermission
          requestedPermission = false;
          tokenProcessed = true;
          const currentTime = Date.now();
          if (currentTime - lastToastTime > 1000) {
            // 1000ms = 1 second
            toast.success("Push notification turned on");
            lastToastTime = currentTime;
          }
          console.log(
            `Subscribed to ${permissionKey}, isSubscribedForPermission: ${isSubscribedForPermission}, permissionGranted: ${permissionGranted}`
          );
        }
      } catch (error) {
        if (requestedPermission) {
          toast.error(
            `Failed to subscribe: ${error.message || "Unknown error"}`
          );
          console.error("Failed to subscribe the user: ", error);
          setTimeout(() => {
            isSubscribedForPermission = false;
          }, 250);
        }
      } finally {
        requestedPermission = false;
        loading = false;
      }
    }
  }, 500);

  /*
  PERSISTENCE TO BACKEND LOGIC
  */
  async function sendSubscriptionToBackend(token) {
    let url =
      addSubscriptionEndpoint +
      `&${notificationTypeQueryParamKey}=` +
      permissionKey;
    const response = await fetch(url, {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
      },
      body: JSON.stringify({ fcm_token: token }),
    });
    if (!response.ok) {
      throw new Error(
        `Failed to add subscription: ${response.status} ${response.statusText}`
      );
    }
  }

  async function removeSubscriptionFromBackend() {
    let url =
      deleteSubscriptionEndpoint +
      `&${notificationTypeQueryParamKey}=` +
      permissionKey;
    const response = await fetch(url, {
      method: "DELETE",
      headers: {
        "Content-Type": "application/json",
      },
      body: JSON.stringify({ fcm_token: token }),
    });
    if (!response.ok) {
      throw new Error(
        `Failed to remove subscription: ${response.status} ${response.statusText}`
      );
    }
  }
</script>

<wc-toast></wc-toast>

{#if loading}
  <PermissionButton {permissionGranted}>
    <LoadingSpinner slot="icon" />
  </PermissionButton>
{:else if permissionGranted && isSubscribedForPermission}
  <PermissionButton on:click={unsubscribeUser} permissionGranted={true}>
    <PushPermissionEnabledIcon slot="icon" />
  </PermissionButton>
{:else}
  <PermissionButton on:click={subscribeUser} permissionGranted={false}>
    <PushPermissionDisabledIcon slot="icon" />
  </PermissionButton>
{/if}
