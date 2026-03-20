<script>
  import { toast } from "wc-toast";

  export let isSubscribed = false;
  export let permissionKey = "";
  export let permissionText = "Specific permission";
  export let subtitleText = "Explain what this permission does";
  export let createPermissionEndpoint = "";
  export let deletePermissionEndpoint = "";
  export let csrfToken = "";

  async function toggleSubscription() {
    if (isSubscribed) {
      await unsubscribeUser();
    } else {
      await subscribeUser();
    }
  }

  function showToast(message) {
    toast.error(message);
  }

  async function subscribeUser() {
    try {
      const response = await fetch(
        createPermissionEndpoint + "?csrf" + csrfToken,
        {
          method: "POST",
          headers: {
            "Content-Type": "application/json",
            "X-CSRF-Token": csrfToken,
          },
          body: JSON.stringify({ permission: permissionKey }),
        }
      );

      if (!response.ok) {
        throw new Error("Failed to subscribe to permission");
      }

      isSubscribed = true;
    } catch (error) {
      console.error("Error subscribing to permission:", error);
      showToast(`Failed to subscribe: ${error.message || "Unknown error"}`);

      setTimeout(() => {
        isSubscribed = false; // Revert the toggle state after 500ms
      }, 250);
    }
  }

  async function unsubscribeUser() {
    try {
      const response = await fetch(
        deletePermissionEndpoint + "?csrf" + csrfToken,
        {
          method: "DELETE",
          headers: {
            "Content-Type": "application/json",
            "X-CSRF-Token": csrfToken,
          },
          body: JSON.stringify({ permission: permissionKey }),
        }
      );

      if (!response.ok) {
        throw new Error("Failed to unsubscribe from permission");
      }

      isSubscribed = false;
    } catch (error) {
      console.error("Error unsubscribing from permission:", error);
      showToast(`Failed to unsubscribe: ${error.message || "Unknown error"}`);

      setTimeout(() => {
        isSubscribed = true; // Revert the toggle state after 500ms
      }, 250);
    }
  }
</script>

<wc-toast></wc-toast>

<div class="bg-slate-100 dark:bg-slate-800 rounded-xl p-2 md:p-4">
  <label class="inline-flex items-center cursor-pointer my-2">
    <input
      type="checkbox"
      bind:checked={isSubscribed}
      on:click={toggleSubscription}
      class="sr-only peer"
    />
    <div
      class="relative min-w-14 h-7 bg-gray-200 peer-focus:outline-none peer-focus:ring-4 peer-focus:ring-blue-300 dark:peer-focus:ring-blue-800 rounded-full peer dark:bg-gray-700 peer-checked:after:translate-x-full rtl:peer-checked:after:-translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-0.5 after:start-[4px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-6 after:w-6 after:transition-all dark:border-gray-600 peer-checked:bg-blue-600"
    ></div>
    <div>
      <div
        class="ms-3 text-sm sm:text-base font-medium text-gray-900 dark:text-gray-300"
      >
        <span>{permissionText}</span>
      </div>
      <div class="ms-3 text-sm font-light text-gray-900 dark:text-gray-300">
        <span>{subtitleText}</span>
      </div>
    </div>
  </label>
</div>
