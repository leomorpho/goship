<script>
  import NotificationPermissions from "../javascript/svelte/components/NotificationPermissions.svelte";

  export let permissionDailyNotif;
  export let permissionPartnerActivity;
  export let vapidPublicKey = "";
  export let subscribedEndpoints = [];
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
</script>

<NotificationPermissions
  {permissionDailyNotif}
  {permissionPartnerActivity}
  {vapidPublicKey}
  {subscribedEndpoints}
  {notificationTypeQueryParamKey}
  {addPushSubscriptionEndpoint}
  {deletePushSubscriptionEndpoint}
  {addFCMPushSubscriptionEndpoint}
  {deleteFCMPushSubscriptionEndpoint}
  {addEmailSubscriptionEndpoint}
  {deleteEmailSubscriptionEndpoint}
  {addSmsSubscriptionEndpoint}
  {deleteSmsSubscriptionEndpoint}
  {phoneSubscriptionEnabled}
/>
