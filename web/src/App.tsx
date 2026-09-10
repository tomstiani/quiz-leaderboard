import { FormEvent, useEffect, useState } from 'react'

type Viewer = {
  role: 'player' | 'owner'
  playerId?: string
  name: string
}

type Dashboard = {
  date: string
  viewer: Viewer
  games: Array<{ id: string; name: string; url: string }>
  players: Array<{
    id: string
    name: string
    completed: number
    combinedScore: number
  }>
}

export function App() {
  const [viewer, setViewer] = useState<Viewer | null | undefined>(undefined)
  const [dashboard, setDashboard] = useState<Dashboard | null>(null)
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
        if (response.ok) setDashboard(await response.json())
        else setError('Could not load today’s leaderboard.')
      })
      .catch(() => setError('Could not load today’s leaderboard.'))
  }, [viewer])

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
              {dashboard.games.map((game) => (
                <a className="game" href={game.url} key={game.id} target="_blank" rel="noreferrer">
                  <strong>{game.name}</strong>
                  <span>Play game →</span>
                </a>
              ))}
            </div>
          </section>

          <section aria-labelledby="leaderboard-heading">
            <h2 id="leaderboard-heading">Scores</h2>
            <div className="table-wrap">
              <table>
                <thead><tr><th>Player</th><th>Completed</th><th>Total</th></tr></thead>
                <tbody>
                  {dashboard.players.map((player) => (
                    <tr key={player.id}>
                      <th>{player.name}</th>
                      <td>{player.completed}/{dashboard.games.length}</td>
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
      <form onSubmit={submit}>
        <p className="eyebrow">Daily Game Leaderboard</p>
        <h1>Welcome back</h1>
        <p>Enter your personal or owner token.</p>
        <label htmlFor="token">Token</label>
        <input id="token" name="token" type="password" autoComplete="current-password" required value={token} onChange={(event) => setToken(event.target.value)} />
        {error && <p role="alert" className="error">{error}</p>}
        <button disabled={submitting}>{submitting ? 'Signing in…' : 'Sign in'}</button>
      </form>
    </main>
  )
}
