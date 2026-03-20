<script>
  import { onMount } from "svelte";
  import { toast } from "wc-toast";

  import PermissionButton from "./PermissionButton.svelte";
  import LoadingSpinner from "./icons/LoadingSpinner.svelte";
  import SmsPermissionDisabledIcon from "./icons/SmsDisabledIcon.svelte";
  import SmsPermissionEnabledIcon from "./icons/SmsEnabledIcon.svelte";

  export let permissionGranted = false;
  export let addSmsSubscriptionEndpoint = "";
  export let deleteSmsSubscriptionEndpoint = "";
  export let permissionKey = "";
  export let notificationTypeQueryParamKey = "";
  export let phoneSubscriptionEnabled = false;

  // TODO: not allowing phone notif changes until we do actually support texting
  phoneSubscriptionEnabled = false;

  let loading = false;

  async function loadDynamicDependencies() {
    if (!window.Swal) {
      await import("https://cdn.jsdelivr.net/npm/sweetalert2@10");
    }
  }

  onMount(async () => {
    await loadDynamicDependencies();
  });

  async function subscribeUser() {
    if (!phoneSubscriptionEnabled) {
      Swal.fire({
        icon: "error",
        title: "Oops...",
        text: "You need to have a verified phone number in your preferences to enable SMS notifications.",
      });

      return;
    }

    loading = true;
    try {
      const response = await fetch(
        addSmsSubscriptionEndpoint +
          `&${notificationTypeQueryParamKey}=` +
          permissionKey,
        {
          method: "POST",
          headers: {
            "Content-Type": "application/json",
          },
        }
      );
      if (!response.ok) {
        throw new Error(
          `Failed to create sms subscription: ${response.status} ${response.statusText}`
        );
      }

      permissionGranted = true;
      toast.success("SMS notification turned on");
    } catch (error) {
      console.error("Failed to subscribe the user: ", error);
      toast.error(`Failed to subscribe: ${error.message || "Unknown error"}`);
    } finally {
      loading = false;
    }
  }

  async function unsubscribeUser() {
    loading = true;
    try {
      const response = await fetch(
        deleteSmsSubscriptionEndpoint +
          `&${notificationTypeQueryParamKey}=` +
          permissionKey,
        {
          method: "DELETE",
          headers: {
            "Content-Type": "application/json",
          },
        }
      );

      if (!response.ok) {
        throw new Error(
          `Failed to delete sms subscription: ${response.status} ${response.statusText}`
        );
      }
      permissionGranted = false;
      toast.success("Email notification turned off");
    } catch (error) {
      console.error("Failed to unsubscribe the user: ", error);
      toast.error(`Failed to unsubscribe: ${error.message || "Unknown error"}`);
    } finally {
      loading = false;
    }
  }
</script>

{#if loading}
  <PermissionButton {permissionGranted}>
    <LoadingSpinner slot="icon" />
  </PermissionButton>
{:else if permissionGranted}
  <PermissionButton on:click={unsubscribeUser} {permissionGranted}>
    <SmsPermissionEnabledIcon slot="icon" />
  </PermissionButton>
{:else}
  <PermissionButton on:click={subscribeUser} {permissionGranted}>
    <SmsPermissionDisabledIcon slot="icon" />
  </PermissionButton>
{/if}
