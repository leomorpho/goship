// Based off of https://github.com/pwa-builder/PWABuilder/blob/main/docs/sw.js

/*
      Welcome to our basic Service Worker! This Service Worker offers a basic offline experience
      while also being easily customizeable. You can add in your own code to implement the capabilities
      listed below, or change anything else you would like.


      Need an introduction to Service Workers? Check our docs here: https://docs.pwabuilder.com/#/home/sw-intro
      Want to learn more about how our Service Worker generation works? Check our docs here: https://docs.pwabuilder.com/#/studio/existing-app?id=add-a-service-worker

      Did you know that Service Workers offer many more capabilities than just offline? 
        - Background Sync: https://microsoft.github.io/win-student-devs/#/30DaysOfPWA/advanced-capabilities/06
        - Periodic Background Sync: https://web.dev/periodic-background-sync/
        - Push Notifications: https://microsoft.github.io/win-student-devs/#/30DaysOfPWA/advanced-capabilities/07?id=push-notifications-on-the-web
        - Badges: https://microsoft.github.io/win-student-devs/#/30DaysOfPWA/advanced-capabilities/07?id=application-badges
    */

const CACHE_NAME = "pwa-cache-v1.2";

const HOSTNAME_WHITELIST = [
  self.location.hostname,
  "fonts.gstatic.com",
  "fonts.googleapis.com",
  "cdn.jsdelivr.net",
];


self.addEventListener("install", (event) => {
  self.skipWaiting();
});

self.addEventListener("message", (event) => {
  if (event.data.action === "skipWaiting") {
    self.skipWaiting();
  }
});

/**
 *  @Lifecycle Activate
 *  New one activated when old isnt being used.
 *
 *  waitUntil(): activating ====> activated
 */
// Clean up old caches during the activate event
self.addEventListener("activate", (event) => {
  const cacheWhitelist = [CACHE_NAME];

  event.waitUntil(
    caches
      .keys()
      .then((cacheNames) => {
        return Promise.all(
          cacheNames.map((cacheName) => {
            if (!cacheWhitelist.includes(cacheName)) {
              return caches.delete(cacheName);
            }
          })
        );
      })
      .then(() => {
        return self.clients.claim();
      })
  );
});

// if (workbox) {
//   console.log(`Yay! Workbox is loaded 🎉`);

//   // Define a list of route patterns
//   const routePatterns = [
//     // "/homeFeed/main",
//     // "/preferences/main",
//     // "/notifications/main",
//     "/profile/main",
//   ];

//   // Custom matching function
//   const matchRoutePatterns = ({ url }) => {
//     return routePatterns.some((pattern) => {
//       if (typeof pattern === "string") {
//         return url.pathname.includes(pattern);
//       } else if (pattern instanceof RegExp) {
//         return pattern.test(url.pathname);
//       }
//       return false;
//     });
//   };

//   // Register the route with the custom strategy
//   workbox.routing.registerRoute(
//     matchRoutePatterns,
//     new workbox.strategies.StaleWhileRevalidate({
//       cacheName: "special-pages",
//       plugins: [
//         new workbox.expiration.ExpirationPlugin({
//           maxEntries: 50,
//         }),
//       ],
//     })
//   );

//   // Other route configurations, using network first and fallback to cache if offline
//   workbox.routing.registerRoute(
//     ({ request }) => request.destination === "document",
//     new workbox.strategies.NetworkFirst({
//       cacheName: "pages",
//       plugins: [
//         new workbox.expiration.ExpirationPlugin({
//           maxEntries: 50,
//         }),
//       ],
//     })
//   );

//   // Cache CSS, JS, and Web Worker requests with a Stale While Revalidate strategy
//   workbox.routing.registerRoute(
//     ({ request }) =>
//       request.destination === "style" ||
//       request.destination === "script" ||
//       request.destination === "worker",
//     new workbox.strategies.StaleWhileRevalidate({
//       cacheName: "assets",
//       plugins: [
//         new workbox.expiration.ExpirationPlugin({
//           maxEntries: 60,
//           maxAgeSeconds: 7 * 24 * 60 * 60, // 7 Days
//         }),
//       ],
//     })
//   );

//   // Cache images with a Cache First strategy
//   workbox.routing.registerRoute(
//     ({ request }) => request.destination === "image",
//     new workbox.strategies.CacheFirst({
//       cacheName: "images",
//       plugins: [
//         new workbox.expiration.ExpirationPlugin({
//           maxEntries: 60,
//           maxAgeSeconds: 7 * 24 * 60 * 60, // 7 Days
//         }),
//       ],
//     })
//   );

//   // Handle HTMX updates
//   self.addEventListener("message", (event) => {
//     if (event.data && event.data.type === "CACHE_UPDATED") {
//       const updatedUrl = new URL(event.data.url, self.location.origin);
//       caches.open("pages").then((cache) => {
//         return fetch(updatedUrl).then((response) => {
//           return cache.put(updatedUrl, response);
//         });
//       });
//     }
//   });
// } else {
//   console.log(`Boo! Workbox didn't load 😬`);
// }

// self.addEventListener("fetch", (event) => {
//   const url = new URL(event.request.url);
//   console.log("SERVICE WORKER", event.request);
//   // Define URLs or patterns to exclude from cache
//   const shouldBypassCache =
//     url.pathname.startsWith("/notifications") ||
//     url.search.includes("no-cache") ||
//     url.search.includes("realtime") ||
//     // This is the path to accept invitations by token.
//     url.pathname.startsWith("/i") ||
//     // Bypass cache for navigations.
//     event.request.mode === "navigate";

//   // Bypass the service worker for non-GET requests.
//   if (event.request.method !== "GET" || shouldBypassCache) {
//     return;
//   }
//   event.respondWith(
//     fetch(event.request).catch(() => {
//       return caches.match(event.request);
//     })
//   );
// });

self.addEventListener("push", (event) => {
  try {
    console.log("[Service Worker] Push Received.");

    const data = event.data.json();
    const title = data.title || "Goship";
    // Extract the unread count from the push message data.
    const message = event.data.json();
    const unreadCount = message.unreadCount;

    // Set or clear the badge on the app icon.
    if (navigator.setAppBadge) {
      if (unreadCount && unreadCount > 0) {
        navigator.setAppBadge(unreadCount);
      } else {
        navigator.clearAppBadge();
      }
    }

    const options = {
      body: data.body,
      icon: "https://chatbond-static.s3.us-west-002.backblazeb2.com/cherie/pwa/manifest-icon-96.maskable.png",
    };

    // It's obligatory to show the notification to the user.
    event.waitUntil(self.registration.showNotification(title, options));
  } catch (error) {
    console.error("Error in push event:", error);
  }
});

self.addEventListener("notificationclick", function (event) {
  // Close the notification when clicked
  event.notification.close();

  // Retrieve dynamic URL from the notification data
  const notificationData = event.notification.data;
  const targetUrl = notificationData
    ? notificationData.url
    : "/auth/notifications";

  // This looks to see if the current is already open and focuses if it is
  event.waitUntil(
    clients
      .matchAll({
        type: "window",
      })
      .then(function (clientList) {
        for (var i = 0; i < clientList.length; i++) {
          var client = clientList[i];
          if (client.url === targetUrl && "focus" in client) {
            return client.focus();
          }
        }
        if (clients.openWindow) {
          return clients.openWindow(targetUrl);
        }
      })
  );
});
