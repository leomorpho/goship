<script>
  import { toast } from "wc-toast";
  import PermissionButton from "./PermissionButton.svelte";
  import EmailPermissionDisabledIcon from "./icons/EmailDisabledIcon.svelte";
  import EmailPermissionEnabledIcon from "./icons/EmailEnabledIcon.svelte";
  import LoadingSpinner from "./icons/LoadingSpinner.svelte";

  export let permissionGranted = false;
  export let addEmailSubscriptionEndpoint = "";
  export let deleteEmailSubscriptionEndpoint = "";
  export let permissionKey = "";
  export let notificationTypeQueryParamKey = "";

  let loading = false;

  async function subscribeUser() {
    loading = true;
    try {
      const response = await fetch(
        addEmailSubscriptionEndpoint +
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
          `Failed to create email subscription: ${response.status} ${response.statusText}`
        );
      }
      permissionGranted = true;
      toast.success("Email notification turned on");
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
        deleteEmailSubscriptionEndpoint +
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
          `Failed to delete email subscription: ${response.status} ${response.statusText}`
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
    <EmailPermissionEnabledIcon slot="icon" />
  </PermissionButton>
{:else}
  <PermissionButton on:click={subscribeUser} {permissionGranted}>
    <EmailPermissionDisabledIcon slot="icon" />
  </PermissionButton>
{/if}
