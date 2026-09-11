import * as stylex from '@stylexjs/stylex'
import { FormEvent, useEffect, useMemo, useState } from 'react'
import { styles } from './styles.stylex'
import type { Game } from './types'

export function SubmissionForm({ game, onConfirmed }: { game: Game; onConfirmed: () => void }) {
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
      <form {...stylex.props(styles.submission)} onSubmit={confirm}>
        {preview && <img {...stylex.props(styles.submissionImage)} src={preview} alt={`${game.name} result preview`} />}
        <label {...stylex.props(styles.submissionLabel)} htmlFor={`score-${game.id}`}>Total score</label>
        <input id={`score-${game.id}`} type="number" min="0" max={game.maxScore || undefined} required value={score} onChange={(event) => setScore(event.target.value)} />
        {error && <p role="alert" {...stylex.props(styles.error)}>{error}</p>}
        <div {...stylex.props(styles.submissionActions)}>
          <button type="button" {...stylex.props(styles.submissionButton)} disabled={busy} aria-busy={busy} onClick={() => { setDraftID(''); setScore(''); setFile(null) }}>Choose another</button>
          <button type="submit" {...stylex.props(styles.submissionButton)} disabled={busy} aria-busy={busy}>{busy ? 'Confirming…' : 'Confirm score'}</button>
        </div>
      </form>
    )
  }

  return (
    <form {...stylex.props(styles.submission)} onSubmit={analyze}>
      <label {...stylex.props(styles.submissionLabel)} htmlFor={`screenshot-${game.id}`}>Result screenshot</label>
      <input {...stylex.props(styles.fileInput)} id={`screenshot-${game.id}`} type="file" accept="image/png,image/jpeg,image/webp" required onChange={(event) => setFile(event.target.files?.[0] ?? null)} />
      <small {...stylex.props(styles.submissionSmall)}>Processed by a third-party vision model.</small>
      {error && <p role="alert" {...stylex.props(styles.error)}>{error}</p>}
      <button type="submit" {...stylex.props(styles.submissionButton)} disabled={!file || busy} aria-busy={busy}>{busy ? 'Analyzing…' : 'Analyze screenshot'}</button>
    </form>
  )
}
