<script lang="ts">
  import { onMount } from "svelte";
  import { writable } from "svelte/store";

  export let vapidPublicKey: string;
  export let checkSubscriptionEndpoint: string = "/check-subscription";
  export let addSubscriptionEndpoint: string = "/subscribe-to-push-notifs";
  export let csrfToken: string;

  let isSubscribed = writable(false);

  onMount(async () => {
    if (!("serviceWorker" in navigator && "PushManager" in window)) {
      console.log("Push notifications are not supported on this platform.");
      return;
    }

    const registration = await navigator.serviceWorker.ready;
    const subscription = await registration.pushManager.getSubscription();

    if (subscription) {
      checkSubscription(subscription);
    } else if (Notification.permission === "default") {
      // Only ask for permission if not already denied or granted
      requestNotificationPermission();
    }
  });

  async function checkSubscription(subscription: any) {
    const response = await fetch(checkSubscriptionEndpoint, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ subscription }),
    });

    const { isSubscribed: subscribed } = await response.json();
    isSubscribed.set(subscribed);

    if (!subscribed) {
      subscribeUser();
    }
  }

  async function requestNotificationPermission() {
    const permission = await Notification.requestPermission();
    if (permission === "granted") {
      subscribeUser();
    }
  }

  async function subscribeUser() {
    const registration = await navigator.serviceWorker.ready;
    try {
      const subscription = await registration.pushManager.subscribe({
        userVisibleOnly: true,
        applicationServerKey: urlB64ToUint8Array(vapidPublicKey),
      });
      await sendSubscriptionToBackend(subscription);
      isSubscribed.set(true);
    } catch (error) {
      console.error("Failed to subscribe to push notifications:", error);
    }
  }

  async function sendSubscriptionToBackend(subscription: any) {
    await fetch(addSubscriptionEndpoint + "?csrf=" + csrfToken, {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
      },
      body: JSON.stringify(subscription),
    });
  }

  function urlB64ToUint8Array(base64String: string) {
    const padding = "=".repeat((4 - (base64String.length % 4)) % 4);
    const base64 = (base64String + padding)
      .replace(/\-/g, "+")
      .replace(/_/g, "/");
    const rawData = window.atob(base64);
    const outputArray = new Uint8Array(rawData.length);

    for (let i = 0; i < rawData.length; ++i) {
      outputArray[i] = rawData.charCodeAt(i);
    }
    return outputArray;
  }
</script>
