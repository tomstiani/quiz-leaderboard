self.addEventListener('push', (event) => {
  const message = event.data?.json() ?? {}
  event.waitUntil(self.registration.showNotification(message.title ?? 'Daily leaderboard', {
    body: message.body,
    data: { url: message.url ?? '/' },
  }))
})

self.addEventListener('notificationclick', (event) => {
  event.notification.close()
  event.waitUntil(clients.matchAll({ type: 'window', includeUncontrolled: true }).then((windows) => {
    const url = event.notification.data?.url ?? '/'
    const existing = windows.find((window) => new URL(window.url).origin === self.location.origin)
    return existing ? existing.navigate(url).then((window) => window?.focus()) : clients.openWindow(url)
  }))
})
