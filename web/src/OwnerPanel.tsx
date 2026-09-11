import * as stylex from '@stylexjs/stylex'
import { FormEvent, useEffect, useState } from 'react'
import { styles } from './styles.stylex'
import type { OwnerSubmission } from './types'

export function OwnerPanel({ refresh, onChanged }: { refresh: number; onChanged: () => void }) {
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
      {error && <p role="alert" {...stylex.props(styles.error)}>{error}</p>}
      {!error && submissions === null && <p role="status">Loading owner controls…</p>}
      {!error && submissions?.length === 0 && <p>No submissions to correct today.</p>}
      <div {...stylex.props(styles.ownerList)}>
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
    <form {...stylex.props(styles.ownerSubmission)} onSubmit={save}>
      <div {...stylex.props(styles.ownerIdentity)}>
        <strong>{submission.playerName}</strong>
        <p {...stylex.props(styles.ownerParagraph)}>{submission.gameName} · {formatScore(submission.normalizedScore)} points · <a href={submission.screenshotUrl} target="_blank" rel="noreferrer">Screenshot</a></p>
      </div>
      <label {...stylex.props(styles.ownerScore)} htmlFor={`owner-score-${submission.id}`}>Raw score
        <input {...stylex.props(styles.ownerScoreInput)} id={`owner-score-${submission.id}`} type="number" min="0" required value={score} onChange={(event) => setScore(event.target.value)} />
      </label>
      {error && <p role="alert" {...stylex.props(styles.error, styles.ownerError)}>{error}</p>}
      <div {...stylex.props(styles.submissionActions, styles.ownerSubmissionActions)}>
        <button type="button" {...stylex.props(styles.ownerButton)} disabled={busy} aria-busy={busy} onClick={reopen}>Reopen</button>
        <button type="submit" {...stylex.props(styles.ownerButton)} disabled={busy} aria-busy={busy}>Save score</button>
      </div>
    </form>
  )
}

function formatScore(score: number) {
  return score.toLocaleString(undefined, { maximumFractionDigits: 1 })
}
