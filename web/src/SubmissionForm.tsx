import * as stylex from '@stylexjs/stylex'
import { ChangeEvent, FormEvent, useEffect, useMemo, useRef, useState } from 'react'
import { styles } from './styles.stylex'
import type { Game } from './types'

export function SubmissionForm({ game, onConfirmed }: { game: Game; onConfirmed: () => void }) {
  const [file, setFile] = useState<File | null>(null)
  const [draftID, setDraftID] = useState('')
  const [score, setScore] = useState('')
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState('')
  const replacementInput = useRef<HTMLInputElement>(null)
  const preview = useMemo(() => file ? URL.createObjectURL(file) : '', [file])

  useEffect(() => () => {
    if (preview) URL.revokeObjectURL(preview)
  }, [preview])

  function chooseScreenshot(event: ChangeEvent<HTMLInputElement>) {
    const screenshot = event.target.files?.[0]
    event.target.value = ''
    if (screenshot) void analyze(screenshot)
  }

  async function analyze(screenshot: File) {
    if (screenshot.size > 10 * 1024 * 1024) {
      setError('Screenshot must be 10 MB or smaller.')
      return
    }
    setFile(screenshot)
    setDraftID('')
    setScore('')
    setBusy(true)
    setError('')
    const body = new FormData()
    body.append('screenshot', screenshot)
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
        <input ref={replacementInput} aria-label="Choose another screenshot" type="file" accept="image/png,image/jpeg,image/webp" disabled={busy} hidden onChange={chooseScreenshot} />
        <div {...stylex.props(styles.submissionActions)}>
          <button type="button" {...stylex.props(styles.submissionButton)} disabled={busy} aria-busy={busy} onClick={() => replacementInput.current?.click()}>Choose another</button>
          <button type="submit" {...stylex.props(styles.submissionButton)} disabled={busy} aria-busy={busy}>{busy ? 'Confirming…' : 'Confirm score'}</button>
        </div>
      </form>
    )
  }

  return (
    <div {...stylex.props(styles.submission)}>
      <small {...stylex.props(styles.submissionSmall)}>Choosing a screenshot sends it to a third-party vision model for analysis.</small>
      {busy ? <p role="status">Analyzing…</p> : <>
        <label {...stylex.props(styles.submissionLabel)} htmlFor={`screenshot-${game.id}`}>Choose result screenshot</label>
        <input {...stylex.props(styles.fileInput)} id={`screenshot-${game.id}`} type="file" accept="image/png,image/jpeg,image/webp" onChange={chooseScreenshot} />
      </>}
      {error && <p role="alert" {...stylex.props(styles.error)}>{error}</p>}
    </div>
  )
}
