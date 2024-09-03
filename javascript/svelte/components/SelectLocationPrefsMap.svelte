<script>
  import {
    DefaultMarker,
    FillLayer,
    FullscreenControl,
    GeoJSON,
    MapLibre,
    NavigationControl,
    ScaleControl,
  } from "svelte-maplibre";
  import { writable } from "svelte/store";

  export let lat = 50;
  export let lon = 50;
  export let rad = 500;
  export let postGeoDataURL = "/post-url";
  let radius = writable(rad);

  // Reactive statement for posting data when lat, lon, or radius changes
  $: if ($latlon && $radius) {
    debouncedPostData(postGeoDataURL, $latlon[1], $latlon[0], $radius);
  }

  // Use a writable store to reactively update the map's center and marker position
  let latlon = writable([lon, lat]);
  // Derived store to create GeoJSON circle based on current latlon and radius
  $: geoJSONCircle = createGeoJSONCircle($latlon, $radius);

  function locateUser() {
    if ("geolocation" in navigator) {
      navigator.geolocation.getCurrentPosition(
        (position) => {
          latlon.set([position.coords.longitude, position.coords.latitude]);
        },
        (error) => {
          console.error("Error getting location:", error);
          alert("Unable to retrieve your location.");
        }
      );
    } else {
      alert("Geolocation is not supported by your browser.");
    }
  }

  // Update latlon only when the marker drag ends
  function onMarkerDragEnd(event) {
    latlon.set(event.detail.lngLat);
  }

  var createGeoJSONCircle = function (center, radiusInKm, points) {
    if (!points) points = 64;

    var coords = {
      latitude: center[1],
      longitude: center[0],
    };

    var km = radiusInKm;

    var ret = [];
    var distanceX = km / (111.32 * Math.cos((coords.latitude * Math.PI) / 180));
    var distanceY = km / 110.574;

    var theta, x, y;
    for (var i = 0; i < points; i++) {
      theta = (i / points) * (2 * Math.PI);
      x = distanceX * Math.cos(theta);
      y = distanceY * Math.sin(theta);

      ret.push([coords.longitude + x, coords.latitude + y]);
    }
    ret.push(ret[0]);

    return {
      type: "Feature",
      geometry: {
        type: "Polygon",
        coordinates: [ret],
      },
    };
  };

  // Debounce function
  function debounce(func, wait) {
    let timeout;
    return function (...args) {
      clearTimeout(timeout);
      timeout = setTimeout(() => {
        func.apply(this, args);
      }, wait);
    };
  }

  async function postData(postURL, latitude, longitude, radius) {
    try {
      const response = await fetch(postURL, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          latitude: latitude,
          longitude: longitude,
          radius: radius,
        }),
      });
      if (!response.ok) {
        // Handle response error
        console.error("Error in posting data", response.statusText);
      } else {
        console.log("Data posted successfully");
      }
    } catch (error) {
      console.error("Error in posting data", error);
    }
  }
  // Creating a debounced version of postData
  const debouncedPostData = debounce(postData, 250);
</script>

<div class="flex flex-col justify-center w-full">
  <div
    class="flex flex-col md:flex-row justify-center items-center w-full my-1"
  >
    <button
      on:click={locateUser}
      class="bg-orange-500 text-white py-3 px-5 rounded-lg mb-5 md:mb-0 mr-2 md:mr-4 flex flex-row whitespace-nowrap"
    >
      <svg
        xmlns="http://www.w3.org/2000/svg"
        viewBox="0 0 24 24"
        fill="currentColor"
        class="w-6 h-6 mr-1"
      >
        <path
          fill-rule="evenodd"
          d="m11.54 22.351.07.04.028.016a.76.76 0 0 0 .723 0l.028-.015.071-.041a16.975 16.975 0 0 0 1.144-.742 19.58 19.58 0 0 0 2.683-2.282c1.944-1.99 3.963-4.98 3.963-8.827a8.25 8.25 0 0 0-16.5 0c0 3.846 2.02 6.837 3.963 8.827a19.58 19.58 0 0 0 2.682 2.282 16.975 16.975 0 0 0 1.145.742ZM12 13.5a3 3 0 1 0 0-6 3 3 0 0 0 0 6Z"
          clip-rule="evenodd"
        />
      </svg>

      Locate Me
    </button>
    <div class="flex-grow flex flex-col justify-center items-center w-full">
      <label
        for="default-range"
        class="block text-sm font-medium text-gray-900 dark:text-white"
        >Max distance: {$radius} km</label
      >
      <div class="py-2 w-full grow">
        <input
          id="default-range"
          type="range"
          min="15"
          max="4000"
          bind:value={$radius}
          class="w-full h-2 bg-gray-200 rounded-lg appearance-none cursor-pointer dark:bg-gray-700"
        />
      </div>
    </div>
  </div>

  <div
    id="alert-3"
    class="flex items-center p-4 mb-4 text-green-800 rounded-lg bg-green-100 dark:bg-gray-800 dark:text-green-400"
    role="alert"
  >
    <svg
      class="flex-shrink-0 w-4 h-4"
      aria-hidden="true"
      xmlns="http://www.w3.org/2000/svg"
      fill="currentColor"
      viewBox="0 0 20 20"
    >
      <path
        d="M10 .5a9.5 9.5 0 1 0 9.5 9.5A9.51 9.51 0 0 0 10 .5ZM9.5 4a1.5 1.5 0 1 1 0 3 1.5 1.5 0 0 1 0-3ZM12 15H8a1 1 0 0 1 0-2h1v-3H8a1 1 0 0 1 0-2h2a1 1 0 0 1 1 1v4h1a1 1 0 0 1 0 2Z"
      />
    </svg>
    <span class="sr-only">Info</span>
    <div class="ms-3 text-sm font-medium">
      You can also click and hold the map marker on the map and move it to your
      desired location.
    </div>
    <button
      type="button"
      class="ms-auto -mx-1.5 -my-1.5 bg-green-50 text-green-500 rounded-lg focus:ring-2 focus:ring-green-400 p-1.5 hover:bg-green-200 inline-flex items-center justify-center h-8 w-8 dark:bg-gray-800 dark:text-green-400 dark:hover:bg-gray-700"
      data-dismiss-target="#alert-3"
      aria-label="Close"
    >
      <span class="sr-only">Close</span>
      <svg
        class="w-3 h-3"
        aria-hidden="true"
        xmlns="http://www.w3.org/2000/svg"
        fill="none"
        viewBox="0 0 14 14"
      >
        <path
          stroke="currentColor"
          stroke-linecap="round"
          stroke-linejoin="round"
          stroke-width="2"
          d="m1 1 6 6m0 0 6 6M7 7l6-6M7 7l-6 6"
        />
      </svg>
    </button>
  </div>

  <MapLibre
    center={$latlon}
    zoom={3}
    class="map rounded-lg w-full"
    style="https://basemaps.cartocdn.com/gl/positron-gl-style/style.json"
  >
    <NavigationControl position="top-left" showCompass={false} />
    <FullscreenControl position="top-left" />
    <ScaleControl />
    <DefaultMarker lngLat={$latlon} draggable on:dragend={onMarkerDragEnd}
    ></DefaultMarker>
    <GeoJSON id="states" data={geoJSONCircle}>
      <FillLayer
        paint={{
          "fill-color": "#008800",
          "fill-opacity": 0.5,
        }}
      />
    </GeoJSON>
  </MapLibre>
</div>

<style>
  :global(.map) {
    height: 500px;
  }
</style>
