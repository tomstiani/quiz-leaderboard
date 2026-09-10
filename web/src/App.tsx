import { FormEvent, useEffect, useMemo, useState } from 'react'

type Viewer = {
  role: 'player' | 'owner'
  playerId?: string
  name: string
}

type Score = {
  gameId: string
  rawScore: number
  normalizedScore: number
  screenshotUrl: string
}

type Game = {
  id: string
  name: string
  url: string
  maxScore: number
}

type OwnerSubmission = {
  id: string
  playerName: string
  gameName: string
  rawScore: number
  normalizedScore: number
  screenshotUrl: string
}

type Dashboard = {
  date: string
  viewer: Viewer
  games: Game[]
  players: Array<{
    id: string
    name: string
    rank: number
    completed: number
    combinedScore: number
    scores: Score[]
  }>
}

export function App() {
  const [viewer, setViewer] = useState<Viewer | null | undefined>(undefined)
  const [dashboard, setDashboard] = useState<Dashboard | null>(null)
  const [refresh, setRefresh] = useState(0)
  const [error, setError] = useState('')

  useEffect(() => {
    fetch('/api/session')
      .then(async (response) => setViewer(response.ok ? await response.json() : null))
      .catch(() => setViewer(null))
  }, [])

  useEffect(() => {
    if (!viewer) return
    fetch('/api/dashboard')
      .then(async (response) => {
        if (response.ok) {
          setDashboard(await response.json())
          setError('')
        } else setError('Could not load today’s leaderboard.')
      })
      .catch(() => setError('Could not load today’s leaderboard.'))
  }, [viewer, refresh])

  useEffect(() => {
    if (!viewer || typeof EventSource === 'undefined') return
    const events = new EventSource('/api/events')
    const update = () => setRefresh((value) => value + 1)
    events.addEventListener('leaderboard', update)
    return () => {
      events.removeEventListener('leaderboard', update)
      events.close()
    }
  }, [viewer])

  if (viewer === undefined) return <main className="loading" role="status" aria-live="polite"><p>Loading…</p></main>
  if (!viewer) return <Login onLogin={setViewer} />

  async function logout() {
    try {
      await removeBrowserPushSubscription().catch(() => {})
      const response = await fetch('/api/logout', { method: 'POST' })
      if (!response.ok) throw new Error()
      setDashboard(null)
      setViewer(null)
    } catch {
      setError('Could not log out. Try again.')
    }
  }

  const currentPlayer = dashboard?.players.find((player) => player.id === viewer.playerId)

  return (
    <main className="dashboard">
      <header>
        <div>
          <p className="eyebrow">{dashboard?.date ?? 'Today'}</p>
          <h1>Daily leaderboard</h1>
          <p>Signed in as {viewer.name}{viewer.role === 'owner' ? ' (owner)' : ''}</p>
        </div>
        <div className="header-actions">
          <BrowserNotifications />
          <button className="secondary" onClick={logout}>Log out</button>
        </div>
      </header>

      {error && <p role="alert" className="error">{error}</p>}
      {!dashboard ? <p>Loading leaderboard…</p> : (
        <>
          <section aria-labelledby="games-heading">
            <h2 id="games-heading">Today’s games</h2>
            <div className="games">
              {dashboard.games.map((game) => {
                const score = currentPlayer?.scores.find((item) => item.gameId === game.id)
                return (
                  <article className="game" key={game.id}>
                    <div className="game-title">
                      <h3>{game.name}</h3>
                      <a href={game.url} target="_blank" rel="noreferrer">Play game →</a>
                    </div>
                    {score ? (
                      <p>Submitted: <b>{score.rawScore}</b> · {formatScore(score.normalizedScore)} points · <a href={score.screenshotUrl} target="_blank" rel="noreferrer">View screenshot</a></p>
                    ) : viewer.role === 'player' ? (
                      <SubmissionForm game={game} onConfirmed={() => setRefresh((value) => value + 1)} />
                    ) : <p>Not submitted</p>}
                  </article>
                )
              })}
            </div>
          </section>

          {viewer.role === 'owner' && <OwnerPanel refresh={refresh} onChanged={() => setRefresh((value) => value + 1)} />}

          <section aria-labelledby="leaderboard-heading">
            <h2 id="leaderboard-heading">Scores</h2>
            <div className="table-wrap">
              <table>
                <thead>
                  <tr>
                    <th>Rank</th>
                    <th>Player</th>
                    <th className="score-detail">Completed</th>
                    {dashboard.games.map((game) => <th className="score-detail" key={game.id}>{game.name}</th>)}
                    <th>Total</th>
                  </tr>
                </thead>
                <tbody>
                  {dashboard.players.map((player) => (
                    <tr key={player.id}>
                      <td>#{player.rank}</td>
                      <th>{player.name}</th>
                      <td className="score-detail">{player.completed}/{dashboard.games.length}</td>
                      {dashboard.games.map((game) => {
                        const score = player.scores.find((item) => item.gameId === game.id)
                        return <td className="score-detail" key={game.id}>{score ? <a href={score.screenshotUrl} target="_blank" rel="noreferrer">{score.rawScore} <small>({formatScore(score.normalizedScore)})</small></a> : '—'}</td>
                      })}
                      <td>{formatScore(player.combinedScore)}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </section>
        </>
      )}
    </main>
  )
}

function BrowserNotifications() {
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

  if (!publicKey) return error ? <span role="alert" className="notification-error">{error}</span> : null

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
    <div className="notification-control">
      <button className="secondary" onClick={subscribed ? disable : enable}>
        {subscribed ? 'Disable notifications' : 'Enable notifications'}
      </button>
      {error && <span role="alert" className="notification-error">{error}</span>}
    </div>
  )
}

function decodeVapidKey(value: string) {
  const base64 = (value + '='.repeat((4 - value.length % 4) % 4)).replaceAll('-', '+').replaceAll('_', '/')
  return Uint8Array.from(atob(base64), (character) => character.charCodeAt(0))
}

async function removeBrowserPushSubscription() {
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

function OwnerPanel({ refresh, onChanged }: { refresh: number; onChanged: () => void }) {
  const [submissions, setSubmissions] = useState<OwnerSubmission[] | null>(null)
  const [error, setError] = useState('')

  useEffect(() => {
    setSubmissions(null)
    fetch('/api/owner/submissions')
      .then(async (response) => {
        if (!response.ok) throw new Error()
        setSubmissions(await response.json())
        setError('')
      })
      .catch(() => setError('Could not load owner controls.'))
  }, [refresh])

  return (
    <section aria-labelledby="owner-heading">
      <h2 id="owner-heading">Owner controls</h2>
      {error && <p role="alert" className="error">{error}</p>}
      {!error && submissions === null && <p role="status">Loading owner controls…</p>}
      {!error && submissions?.length === 0 && <p>No submissions to correct today.</p>}
      <div className="owner-list">
        {submissions?.map((submission) => (
          <OwnerSubmissionForm key={submission.id} submission={submission} onChanged={onChanged} />
        ))}
      </div>
    </section>
  )
}

function OwnerSubmissionForm({ submission, onChanged }: { submission: OwnerSubmission; onChanged: () => void }) {
  const [score, setScore] = useState(submission.rawScore.toString())
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState('')

  async function save(event: FormEvent) {
    event.preventDefault()
    setBusy(true)
    setError('')
    try {
      const response = await fetch(`/api/owner/submissions/${submission.id}/score`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ score: Number(score) }),
      })
      if (!response.ok) throw new Error((await response.text()).trim())
      onChanged()
    } catch (reason) {
      setError(reason instanceof Error && reason.message ? reason.message : 'Could not correct score.')
    } finally {
      setBusy(false)
    }
  }

  async function reopen() {
    if (!window.confirm(`Remove ${submission.playerName}’s ${submission.gameName} submission and allow another upload?`)) return
    setBusy(true)
    setError('')
    try {
      const response = await fetch(`/api/owner/submissions/${submission.id}`, { method: 'DELETE' })
      if (!response.ok) throw new Error((await response.text()).trim())
      onChanged()
    } catch (reason) {
      setError(reason instanceof Error && reason.message ? reason.message : 'Could not reopen submission.')
    } finally {
      setBusy(false)
    }
  }

  return (
    <form className="owner-submission" onSubmit={save}>
      <div className="owner-identity">
        <strong>{submission.playerName}</strong>
        <p>{submission.gameName} · {formatScore(submission.normalizedScore)} points · <a href={submission.screenshotUrl} target="_blank" rel="noreferrer">Screenshot</a></p>
      </div>
      <label className="owner-score" htmlFor={`owner-score-${submission.id}`}>Raw score
        <input id={`owner-score-${submission.id}`} type="number" min="0" required value={score} onChange={(event) => setScore(event.target.value)} />
      </label>
      {error && <p role="alert" className="error">{error}</p>}
      <div className="submission-actions">
        <button type="button" className="secondary" disabled={busy} aria-busy={busy} onClick={reopen}>Reopen</button>
        <button type="submit" disabled={busy} aria-busy={busy}>Save score</button>
      </div>
    </form>
  )
}

function formatScore(score: number) {
  return score.toLocaleString(undefined, { maximumFractionDigits: 1 })
}

function SubmissionForm({ game, onConfirmed }: { game: Game; onConfirmed: () => void }) {
  const [file, setFile] = useState<File | null>(null)
  const [draftID, setDraftID] = useState('')
  const [score, setScore] = useState('')
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState('')
  const preview = useMemo(() => file ? URL.createObjectURL(file) : '', [file])

  useEffect(() => () => {
    if (preview) URL.revokeObjectURL(preview)
  }, [preview])

  async function analyze(event: FormEvent) {
    event.preventDefault()
    if (!file) return
    if (file.size > 10 * 1024 * 1024) {
      setError('Screenshot must be 10 MB or smaller.')
      return
    }
    setBusy(true)
    setError('')
    const body = new FormData()
    body.append('screenshot', file)
    try {
      const response = await fetch(`/api/games/${game.id}/draft`, { method: 'POST', body })
      if (!response.ok) {
        setError((await response.text()).trim() || 'Could not analyze screenshot.')
        return
      }
      const draft: { id: string; score: number | null } = await response.json()
      setDraftID(draft.id)
      setScore(draft.score?.toString() ?? '')
    } catch {
      setError('Could not reach the server.')
    } finally {
      setBusy(false)
    }
  }

  async function confirm(event: FormEvent) {
    event.preventDefault()
    setBusy(true)
    setError('')
    try {
      const response = await fetch(`/api/drafts/${draftID}/confirm`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ score: Number(score) }),
      })
      if (!response.ok) {
        setError((await response.text()).trim() || 'Could not confirm score.')
        return
      }
      onConfirmed()
    } catch {
      setError('Could not reach the server.')
    } finally {
      setBusy(false)
    }
  }

  if (draftID) {
    return (
      <form className="submission" onSubmit={confirm}>
        {preview && <img src={preview} alt={`${game.name} result preview`} />}
        <label htmlFor={`score-${game.id}`}>Total score</label>
        <input id={`score-${game.id}`} type="number" min="0" max={game.maxScore || undefined} required value={score} onChange={(event) => setScore(event.target.value)} />
        {error && <p role="alert" className="error">{error}</p>}
        <div className="submission-actions">
          <button type="button" className="secondary" disabled={busy} aria-busy={busy} onClick={() => { setDraftID(''); setScore(''); setFile(null) }}>Choose another</button>
          <button type="submit" disabled={busy} aria-busy={busy}>{busy ? 'Confirming…' : 'Confirm score'}</button>
        </div>
      </form>
    )
  }

  return (
    <form className="submission" onSubmit={analyze}>
      <label htmlFor={`screenshot-${game.id}`}>Result screenshot</label>
      <input id={`screenshot-${game.id}`} type="file" accept="image/png,image/jpeg,image/webp" required onChange={(event) => setFile(event.target.files?.[0] ?? null)} />
      <small>Processed by a third-party vision model.</small>
      {error && <p role="alert" className="error">{error}</p>}
      <button type="submit" disabled={!file || busy} aria-busy={busy}>{busy ? 'Analyzing…' : 'Analyze screenshot'}</button>
    </form>
  )
}

function Login({ onLogin }: { onLogin: (viewer: Viewer) => void }) {
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
    <main className="login">
      <form className="login-form" onSubmit={submit}>
        <p className="eyebrow">Daily Game Leaderboard</p>
        <h1>Welcome back</h1>
        <p>Enter your personal or owner token.</p>
        <label htmlFor="token">Token</label>
        <input id="token" name="token" type="password" autoComplete="current-password" required value={token} onChange={(event) => setToken(event.target.value)} />
        {error && <p role="alert" className="error">{error}</p>}
        <button type="submit" disabled={submitting} aria-busy={submitting}>{submitting ? 'Signing in…' : 'Sign in'}</button>
      </form>
    </main>
  )
}
