import * as stylex from '@stylexjs/stylex'
import { FormEvent, useState } from 'react'
import { styles } from './styles.stylex'
import type { Viewer } from './types'

export function Login({ onLogin }: { onLogin: (viewer: Viewer) => void }) {
  const [token, setToken] = useState('')
  const [error, setError] = useState('')
  const [submitting, setSubmitting] = useState(false)

  async function submit(event: FormEvent) {
    event.preventDefault()
    setSubmitting(true)
    setError('')
    try {
      const response = await fetch('/api/login', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ token }),
      })
      if (!response.ok) {
        setError(response.status === 401 ? 'That token is not valid.' : 'Could not sign in.')
        return
      }
      onLogin(await response.json())
    } catch {
      setError('Could not reach the server.')
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <main {...stylex.props(styles.main, styles.login)}>
      <form {...stylex.props(styles.loginForm)} onSubmit={submit}>
        <p {...stylex.props(styles.eyebrow)}>Daily Game Leaderboard</p>
        <h1 {...stylex.props(styles.loginHeading)}>Welcome back</h1>
        <p>Enter your personal or owner token.</p>
        <label htmlFor="token">Token</label>
        <input id="token" name="token" type="password" autoComplete="current-password" required value={token} onChange={(event) => setToken(event.target.value)} />
        {error && <p role="alert" {...stylex.props(styles.error)}>{error}</p>}
        <button type="submit" {...stylex.props(styles.loginButton)} disabled={submitting} aria-busy={submitting}>{submitting ? 'Signing in…' : 'Sign in'}</button>
      </form>
    </main>
  )
}
