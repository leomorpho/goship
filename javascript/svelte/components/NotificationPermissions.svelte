<script>
  import { onMount } from "svelte";
  import EmailSubscribe from "./notifications/EmailSubscribe.svelte";
  import IosSubscribePush from "./notifications/IOSSubscribePush.svelte";
  import PwaSubscribePush from "./notifications/PwaSubscribePush.svelte";

  export let permissionDailyNotif;
  export let permissionPartnerActivity;
  export let vapidPublicKey = "";
  export let subscribedEndpoints = []; // Array of subscribed endpoints passed from the server
  export let notificationTypeQueryParamKey = "notifType";

  export let addPushSubscriptionEndpoint = "";
  export let deletePushSubscriptionEndpoint = "";
  export let addFCMPushSubscriptionEndpoint = "";
  export let deleteFCMPushSubscriptionEndpoint = "";

  export let addEmailSubscriptionEndpoint = "";
  export let deleteEmailSubscriptionEndpoint = "";
  export let addSmsSubscriptionEndpoint = "";
  export let deleteSmsSubscriptionEndpoint = "";
  export let phoneSubscriptionEnabled = false;

  let isPwaPushNotificationsPossibleBoolean = false;
  let isIosNativeAppBoolean = false;

  function hasPushGranted(platform, platformsList) {
    return platformsList.some((p) => p.platform === platform && p.granted);
  }

  console.log(permissionDailyNotif);
  console.log(permissionPartnerActivity);

  async function isPwaPushNotificationsPossible() {
    if ("serviceWorker" in navigator && navigator.serviceWorker.ready) {
      console.log(
        "isPwaPushNotificationsPossible, serviceworker in navigator",
        true
      );

      const registration = await navigator.serviceWorker.ready;
      if (registration.pushManager) {
        console.log(
          "isPwaPushNotificationsPossible registration.pushManager",
          true
        );
        return true;
      }
    }
    console.log("isPwaPushNotificationsPossible", false);
    return false;
  }

  onMount(async () => {
    isPwaPushNotificationsPossibleBoolean =
      await isPwaPushNotificationsPossible();

    if (
      window.webkit &&
      window.webkit.messageHandlers &&
      window.webkit.messageHandlers["push-permission-request"] &&
      window.webkit.messageHandlers["push-permission-state"]
    ) {
      isIosNativeAppBoolean = true;
    }

    console.log(
      "isPwaPushNotificationsPossibleBoolean",
      isPwaPushNotificationsPossibleBoolean
    );
    console.log("isIosNativeAppBoolean", isIosNativeAppBoolean);
  });
</script>

<wc-toast></wc-toast>

<div class="bg-slate-100 dark:bg-slate-800 rounded-xl p-2 md:p-4 my-2">
  <div class="inline-flex items-center cursor-pointer my-2">
    <div>
      <div
        class="ms-3 text-sm sm:text-base font-medium text-gray-900 dark:text-gray-300"
      >
        <span>{permissionDailyNotif.title}</span>
      </div>
      <div class="ms-3 text-sm font-light text-gray-900 dark:text-gray-300">
        <span>{permissionDailyNotif.subtitle}</span>
      </div>
    </div>
  </div>

  <div class="flex flex-row justify-center">
    {#if isPwaPushNotificationsPossibleBoolean}
      <PwaSubscribePush
        {vapidPublicKey}
        {subscribedEndpoints}
        addSubscriptionEndpoint={addPushSubscriptionEndpoint}
        deleteSubscriptionEndpoint={deletePushSubscriptionEndpoint}
        {notificationTypeQueryParamKey}
        permissionKey={permissionDailyNotif.permission}
        permissionGranted={hasPushGranted(
          "push",
          permissionDailyNotif.platforms_list
        )}
      />
    {:else if isIosNativeAppBoolean}
      <IosSubscribePush
        addSubscriptionEndpoint={addFCMPushSubscriptionEndpoint}
        deleteSubscriptionEndpoint={deleteFCMPushSubscriptionEndpoint}
        {notificationTypeQueryParamKey}
        permissionKey={permissionDailyNotif.permission}
        permissionGranted={hasPushGranted(
          "fcm_push",
          permissionDailyNotif.platforms_list
        )}
      />
    {/if}
    <EmailSubscribe
      {notificationTypeQueryParamKey}
      permissionKey={permissionDailyNotif.permission}
      {addEmailSubscriptionEndpoint}
      {deleteEmailSubscriptionEndpoint}
      permissionGranted={hasPushGranted(
        "email",
        permissionDailyNotif.platforms_list
      )}
    />
    <!-- <SmsSubscribe
      {phoneSubscriptionEnabled}
      {notificationTypeQueryParamKey}
      permissionKey={permissionDailyNotif.permission}
      {addSmsSubscriptionEndpoint}
      {deleteSmsSubscriptionEndpoint}
      permissionGranted={hasPushGranted(
        "sms",
        permissionDailyNotif.platforms_list
      )}
    /> -->
  </div>
</div>

<div class="bg-slate-100 dark:bg-slate-800 rounded-xl p-2 md:p-4 my-2">
  <div class="inline-flex items-center cursor-pointer my-2">
    <div>
      <div
        class="ms-3 text-sm sm:text-base font-medium text-gray-900 dark:text-gray-300"
      >
        <span>{permissionPartnerActivity.title}</span>
      </div>
      <div class="ms-3 text-sm font-light text-gray-900 dark:text-gray-300">
        <span>{permissionPartnerActivity.subtitle}</span>
      </div>
    </div>
  </div>
  <div class="flex flex-row justify-center">
    {#if isPwaPushNotificationsPossibleBoolean}
      <PwaSubscribePush
        {vapidPublicKey}
        {subscribedEndpoints}
        addSubscriptionEndpoint={addPushSubscriptionEndpoint}
        deleteSubscriptionEndpoint={deletePushSubscriptionEndpoint}
        {notificationTypeQueryParamKey}
        permissionKey={permissionPartnerActivity.permission}
        permissionGranted={hasPushGranted(
          "push",
          permissionPartnerActivity.platforms_list
        )}
      />
    {:else if isIosNativeAppBoolean}
      <IosSubscribePush
        addSubscriptionEndpoint={addFCMPushSubscriptionEndpoint}
        deleteSubscriptionEndpoint={deleteFCMPushSubscriptionEndpoint}
        {notificationTypeQueryParamKey}
        permissionKey={permissionPartnerActivity.permission}
        permissionGranted={hasPushGranted(
          "fcm_push",
          permissionPartnerActivity.platforms_list
        )}
      />
    {/if}
    <EmailSubscribe
      {notificationTypeQueryParamKey}
      permissionKey={permissionPartnerActivity.permission}
      {addEmailSubscriptionEndpoint}
      {deleteEmailSubscriptionEndpoint}
      permissionGranted={hasPushGranted(
        "email",
        permissionPartnerActivity.platforms_list
      )}
    />
    <!-- <SmsSubscribe
      {phoneSubscriptionEnabled}
      {notificationTypeQueryParamKey}
      permissionKey={permissionPartnerActivity.permission}
      {addSmsSubscriptionEndpoint}
      {deleteSmsSubscriptionEndpoint}
      permissionGranted={hasPushGranted(
        "sms",
        permissionPartnerActivity.platforms_list
      )}
    /> -->
  </div>
</div>
