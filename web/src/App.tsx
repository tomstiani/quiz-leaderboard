import * as stylex from '@stylexjs/stylex'
import { useEffect, useState } from 'react'
import { BrowserNotifications, removeBrowserPushSubscription } from './BrowserNotifications'
import { Login } from './Login'
import { OwnerPanel } from './OwnerPanel'
import { SubmissionForm } from './SubmissionForm'
import { styles } from './styles.stylex'
import type { Game, Viewer } from './types'

type Score = {
  gameId: string
  rawScore: number
  normalizedScore: number
  screenshotUrl: string
}

type Dashboard = {
  date: string
  week: string[]
  viewer: Viewer
  games: Game[]
  players: Array<{
    id: string
    name: string
    rank: number
    completed: number
    combinedScore: number
    scores: Score[]
    dailyTotals: Record<string, number>
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

  if (viewer === undefined) return <main {...stylex.props(styles.main, styles.loading)} role="status" aria-live="polite"><p>Loading…</p></main>
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
    <main {...stylex.props(styles.main, styles.dashboard)}>
      <header {...stylex.props(styles.header)}>
        <div>
          <p {...stylex.props(styles.eyebrow)}>{dashboard?.date ?? 'Today'}</p>
          <h1>Daily leaderboard</h1>
          <p>Signed in as {viewer.name}{viewer.role === 'owner' ? ' (owner)' : ''}</p>
        </div>
        <div {...stylex.props(styles.headerActions)}>
          <BrowserNotifications />
          <button {...stylex.props(styles.headerButton)} onClick={logout}>Log out</button>
        </div>
      </header>

      {error && <p role="alert" {...stylex.props(styles.error)}>{error}</p>}
      {!dashboard ? <p>Loading leaderboard…</p> : (
        <>
          <section aria-labelledby="games-heading">
            <h2 id="games-heading">Today’s games</h2>
            <div {...stylex.props(styles.games)}>
              {dashboard.games.map((game) => {
                const score = currentPlayer?.scores.find((item) => item.gameId === game.id)
                return (
                  <article {...stylex.props(styles.game)} key={game.id}>
                    <div {...stylex.props(styles.gameTitle)}>
                      <h3>{game.name}</h3>
                      <a {...stylex.props(styles.gameTitleLink)} href={game.url} target="_blank" rel="noreferrer">Play game →</a>
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
            <div {...stylex.props(styles.tableWrap)}>
              <table>
                <thead>
                  <tr>
                    <th>Rank</th>
                    <th>Player</th>
                    <th {...stylex.props(styles.scoreDetail)}>Completed</th>
                    {dashboard.games.map((game) => <th {...stylex.props(styles.scoreDetail)} key={game.id}>{game.name}</th>)}
                    <th>Total</th>
                  </tr>
                </thead>
                <tbody>
                  {dashboard.players.map((player) => (
                    <tr key={player.id}>
                      <td>#{player.rank}</td>
                      <th>{player.name}</th>
                      <td {...stylex.props(styles.scoreDetail)}>{player.completed}/{dashboard.games.length}</td>
                      {dashboard.games.map((game) => {
                        const score = player.scores.find((item) => item.gameId === game.id)
                        return <td {...stylex.props(styles.scoreDetail)} key={game.id}>{score ? <a href={score.screenshotUrl} target="_blank" rel="noreferrer">{score.rawScore} <small>({formatScore(score.normalizedScore)})</small></a> : '—'}</td>
                      })}
                      <td>{formatScore(player.combinedScore)}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </section>

          <section aria-labelledby="week-heading">
            <h2 id="week-heading">Week overview</h2>
            <div {...stylex.props(styles.tableWrap)}>
              <table>
                <thead>
                  <tr>
                    <th>Player</th>
                    {dashboard.week.map((date) => <th key={date} aria-current={date === dashboard.date ? 'date' : undefined}>{formatDay(date)}</th>)}
                  </tr>
                </thead>
                <tbody>
                  {dashboard.players.map((player) => (
                    <tr key={player.id}>
                      <th>{player.name}</th>
                      {dashboard.week.map((date) => <td key={date}>{player.dailyTotals[date] === undefined ? '—' : formatScore(player.dailyTotals[date])}</td>)}
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

function formatScore(score: number) {
  return score.toLocaleString(undefined, { maximumFractionDigits: 1 })
}

function formatDay(date: string) {
  return new Date(`${date}T00:00:00`).toLocaleDateString(undefined, { weekday: 'short', day: 'numeric' })
}
