import { toast } from "wc-toast";
import MultiSelectComponent from "./components/MultiSelectComponent.svelte";
import NotificationPermissions from "./components/NotificationPermissions.svelte";
import PhoneNumberPicker from "./components/PhoneNumberPicker.svelte";
import PhotoUploader from "./components/PhotoUploader.svelte";
import PwaInstallButton from "./components/PwaInstallButton.svelte";
import SingleSelect from "./components/SingleSelect.svelte";
import ThemeToggle from "./components/ThemeToggle.svelte";
import PwaSubscribePush from "./components/notifications/PwaSubscribePush.svelte";

// Define a registry object that maps names to Svelte component classes
const SvelteComponentRegistry = {
  MultiSelectComponent,
  // Mostly from https://github.com/flo-bit/svelte-swiper-cards
  PhotoUploader,
  SingleSelect,
  PhoneNumberPicker,
  PwaInstallButton,
  PwaSubscribePush,
  ThemeToggle,
  NotificationPermissions,
};

// Assuming `window.svelteInstances` is a map to track component instances
window.svelteInstances = window.svelteInstances || {};

// Utility function to render any Svelte component by its registry name and ID
function renderSvelteComponentByName(componentName, id, props = {}) {
  const Component = SvelteComponentRegistry[componentName];
  if (!Component) {
    throw new Error(`Component ${componentName} not found in registry`);
  }

  const rootElement = document.getElementById(id);
  if (!rootElement) {
    throw new Error(`Could not find element with id ${id}`);
  }
  // Clear existing content at the root element to prevent duplication
  rootElement.innerHTML = "";

  // Destroy the existing instance if it exists
  if (window.svelteInstances[id]) {
    window.svelteInstances[id].$destroy();
    window.svelteInstances[id] = null; // Clear the reference after destruction
  }

  // Instantiate the new Svelte component with the target and props
  window.svelteInstances[id] = new Component({
    target: rootElement,
    props: props,
  });
}

window.renderSvelteComponent = function (componentName, id, props = {}) {
  renderSvelteComponentByName(componentName, id, props);
};

window.successToast = function (text, timeToShow = 1000) {
  toast.success(text, { duration: timeToShow });
};

window.errorToast = function (text, timeToShow = 1000) {
  toast.error(text, { duration: timeToShow });
};
