import * as stylex from '@stylexjs/stylex'
import { useEffect, useState } from 'react'
import { styles } from './styles.stylex'

export function BrowserNotifications() {
  const [publicKey, setPublicKey] = useState('')
  const [subscribed, setSubscribed] = useState(false)
  const [error, setError] = useState('')

  useEffect(() => {
    if (!('serviceWorker' in navigator) || !('PushManager' in window) || !('Notification' in window)) return
    fetch('/api/push/config')
      .then(async (response) => {
        if (!response.ok) throw new Error()
        const config = await response.json()
        if (!config.enabled) return
        setPublicKey(config.publicKey)
        const registration = await navigator.serviceWorker.register('/sw.js')
        setSubscribed(Boolean(await registration.pushManager.getSubscription()))
      })
      .catch(() => setError('Browser notifications are unavailable.'))
  }, [])

  if (!publicKey) return error ? <span role="alert" {...stylex.props(styles.notificationError)}>{error}</span> : null

  async function enable() {
    setError('')
    try {
      if (await Notification.requestPermission() !== 'granted') {
        setError('Notifications were not allowed.')
        return
      }
      const registration = await navigator.serviceWorker.ready
      const subscription = await registration.pushManager.subscribe({
        userVisibleOnly: true,
        applicationServerKey: decodeVapidKey(publicKey),
      })
      const response = await fetch('/api/push/subscriptions', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(subscription),
      })
      if (!response.ok) throw new Error()
      setSubscribed(true)
    } catch {
      setError('Could not enable browser notifications.')
    }
  }

  async function disable() {
    setError('')
    try {
      await removeBrowserPushSubscription()
      setSubscribed(false)
    } catch {
      setError('Could not disable browser notifications.')
    }
  }

  return (
    <div {...stylex.props(styles.notificationControl)}>
      <button {...stylex.props(styles.notificationButton)} onClick={subscribed ? disable : enable}>
        {subscribed ? 'Disable notifications' : 'Enable notifications'}
      </button>
      {error && <span role="alert" {...stylex.props(styles.notificationError)}>{error}</span>}
    </div>
  )
}

function decodeVapidKey(value: string) {
  const base64 = (value + '='.repeat((4 - value.length % 4) % 4)).replaceAll('-', '+').replaceAll('_', '/')
  return Uint8Array.from(atob(base64), (character) => character.charCodeAt(0))
}

export async function removeBrowserPushSubscription() {
  if (!('serviceWorker' in navigator) || !('PushManager' in window)) return
  const registration = await navigator.serviceWorker.getRegistration()
  const subscription = await registration?.pushManager.getSubscription()
  if (!subscription) return
  const response = await fetch('/api/push/subscriptions', {
    method: 'DELETE',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ endpoint: subscription.endpoint }),
  })
  if (!response.ok) throw new Error()
  await subscription.unsubscribe()
}
