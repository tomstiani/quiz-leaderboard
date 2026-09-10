import { FormEvent, useEffect, useMemo, useState } from 'react'

type Viewer = {
  role: 'player' | 'owner'
  playerId?: string
  name: string
}

type Score = {
  gameId: string
  rawScore: number
  screenshotUrl: string
}

type Game = {
  id: string
  name: string
  url: string
  maxScore: number
}

type Dashboard = {
  date: string
  viewer: Viewer
  games: Game[]
  players: Array<{
    id: string
    name: string
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

  if (viewer === undefined) return <main><p>Loading…</p></main>
  if (!viewer) return <Login onLogin={setViewer} />

  async function logout() {
    try {
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
        <button className="secondary" onClick={logout}>Log out</button>
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
                      <strong>{game.name}</strong>
                      <a href={game.url} target="_blank" rel="noreferrer">Play game →</a>
                    </div>
                    {score ? (
                      <p>Submitted: <b>{score.rawScore}</b> · <a href={score.screenshotUrl} target="_blank" rel="noreferrer">View screenshot</a></p>
                    ) : viewer.role === 'player' ? (
                      <SubmissionForm game={game} onConfirmed={() => setRefresh((value) => value + 1)} />
                    ) : <p>Not submitted</p>}
                  </article>
                )
              })}
            </div>
          </section>

          <section aria-labelledby="leaderboard-heading">
            <h2 id="leaderboard-heading">Scores</h2>
            <div className="table-wrap">
              <table>
                <thead>
                  <tr>
                    <th>Player</th>
                    <th>Completed</th>
                    {dashboard.games.map((game) => <th key={game.id}>{game.name}</th>)}
                    <th>Total</th>
                  </tr>
                </thead>
                <tbody>
                  {dashboard.players.map((player) => (
                    <tr key={player.id}>
                      <th>{player.name}</th>
                      <td>{player.completed}/{dashboard.games.length}</td>
                      {dashboard.games.map((game) => {
                        const score = player.scores.find((item) => item.gameId === game.id)
                        return <td key={game.id}>{score ? <a href={score.screenshotUrl} target="_blank" rel="noreferrer">{score.rawScore}</a> : '—'}</td>
                      })}
                      <td>{player.combinedScore}</td>
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
          <button type="button" className="secondary" disabled={busy} onClick={() => { setDraftID(''); setScore(''); setFile(null) }}>Choose another</button>
          <button type="submit" disabled={busy}>{busy ? 'Confirming…' : 'Confirm score'}</button>
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
      <button type="submit" disabled={!file || busy}>{busy ? 'Analyzing…' : 'Analyze screenshot'}</button>
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
        <button type="submit" disabled={submitting}>{submitting ? 'Signing in…' : 'Sign in'}</button>
      </form>
    </main>
  )
}
