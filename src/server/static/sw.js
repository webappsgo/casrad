// CASRAD Service Worker — offline support per AI.md PART 16
const CACHE_VERSION = 'casrad-v1';
const STATIC_ASSETS = [
  '/',
  '/static/css/theme.css',
  '/static/manifest.json',
];

// Install: pre-cache static assets
self.addEventListener('install', function(event) {
  event.waitUntil(
    caches.open(CACHE_VERSION).then(function(cache) {
      return cache.addAll(STATIC_ASSETS);
    })
  );
  self.skipWaiting();
});

// Activate: remove stale caches
self.addEventListener('activate', function(event) {
  event.waitUntil(
    caches.keys().then(function(keys) {
      return Promise.all(
        keys.filter(function(key) { return key !== CACHE_VERSION; })
            .map(function(key) { return caches.delete(key); })
      );
    })
  );
  self.clients.claim();
});

// Fetch: network-first for API, cache-first for static assets
self.addEventListener('fetch', function(event) {
  var url = new URL(event.request.url);

  // Always fetch API and stream requests from network
  if (url.pathname.startsWith('/api/') ||
      url.pathname.startsWith('/stream/') ||
      url.pathname.startsWith('/subsonic/') ||
      url.pathname.startsWith('/ampache/') ||
      event.request.method !== 'GET') {
    return;
  }

  // Cache-first for static assets
  if (url.pathname.startsWith('/static/')) {
    event.respondWith(
      caches.match(event.request).then(function(cached) {
        return cached || fetch(event.request).then(function(response) {
          var clone = response.clone();
          caches.open(CACHE_VERSION).then(function(cache) {
            cache.put(event.request, clone);
          });
          return response;
        });
      })
    );
    return;
  }

  // Network-first for HTML pages with offline fallback
  event.respondWith(
    fetch(event.request).catch(function() {
      return caches.match(event.request).then(function(cached) {
        return cached || caches.match('/');
      });
    })
  );
});
