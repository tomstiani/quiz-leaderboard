// @vitest-environment jsdom

import { cleanup, fireEvent, render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { App } from './App'

const player = { role: 'player', playerId: 'alice', name: 'Alice' }
const dashboard = {
  date: '2026-09-10',
  viewer: player,
  games: [
    { id: 'geopolitix', name: 'Geopolitix', url: 'https://geopolitix.live/', maxScore: 900 },
    { id: 'krillion', name: 'Krillion', url: 'https://krillion.io/', maxScore: 700 },
  ],
  players: [
    { id: 'alice', name: 'Alice', rank: 1, completed: 0, combinedScore: 0, scores: [] },
    { id: 'bob', name: 'Bob', rank: 1, completed: 0, combinedScore: 0, scores: [] },
  ],
}

afterEach(() => {
  cleanup()
  vi.restoreAllMocks()
  vi.unstubAllGlobals()
})

describe('App', () => {
  it('signs in and displays the configured dashboard', async () => {
    const fetchMock = vi.fn(async (input: string | URL | Request, init?: RequestInit) => {
      const path = input.toString()
      if (path === '/api/session') return new Response('', { status: 401 })
      if (path === '/api/login') {
        expect(JSON.parse(init?.body as string)).toEqual({ token: 'friend-token' })
        return Response.json(player)
      }
      if (path === '/api/dashboard') return Response.json(dashboard)
      throw new Error(`unexpected request: ${path}`)
    })
    vi.stubGlobal('fetch', fetchMock)
    const user = userEvent.setup()
    render(<App />)

    await user.type(await screen.findByLabelText('Token'), 'friend-token')
    await user.click(screen.getByRole('button', { name: 'Sign in' }))

    expect(await screen.findByRole('heading', { name: 'Daily leaderboard' })).toBeTruthy()
    expect(screen.getAllByRole('link', { name: 'Play game →' })[0].getAttribute('href')).toBe('https://geopolitix.live/')
    expect(screen.getByText('Alice')).toBeTruthy()
    expect(screen.getByText('Bob')).toBeTruthy()
    expect(screen.getAllByText('0/2')).toHaveLength(2)
  })

  it('uploads, reviews, edits, and confirms a screenshot', async () => {
    const confirmedDashboard = {
      ...dashboard,
      players: [
        { ...dashboard.players[0], completed: 1, combinedScore: 33.333333, scores: [{ gameId: 'geopolitix', rawScore: 300, normalizedScore: 33.333333, screenshotUrl: '/api/screenshots/draft-1' }] },
        dashboard.players[1],
      ],
    }
    let dashboardCalls = 0
    const fetchMock = vi.fn(async (input: string | URL | Request, init?: RequestInit) => {
      const path = input.toString()
      if (path === '/api/session') return Response.json(player)
      if (path === '/api/dashboard') return Response.json(dashboardCalls++ ? confirmedDashboard : dashboard)
      if (path === '/api/games/geopolitix/draft') {
        expect(init?.body).toBeInstanceOf(FormData)
        return Response.json({ id: 'draft-1', score: 321 }, { status: 201 })
      }
      if (path === '/api/drafts/draft-1/confirm') {
        expect(JSON.parse(init?.body as string)).toEqual({ score: 300 })
        return Response.json({ id: 'draft-1', score: 300 })
      }
      throw new Error(`unexpected request: ${path}`)
    })
    vi.stubGlobal('fetch', fetchMock)
    URL.createObjectURL = vi.fn(() => 'blob:preview')
    URL.revokeObjectURL = vi.fn()
    const user = userEvent.setup()
    render(<App />)

    const file = new File([new Uint8Array([137, 80, 78, 71])], 'score.png', { type: 'image/png' })
    await user.upload((await screen.findAllByLabelText('Result screenshot'))[0], file)
    fireEvent.submit(screen.getAllByRole('button', { name: 'Analyze screenshot' })[0].closest('form')!)
    await waitFor(() => expect(fetchMock.mock.calls.some(([input]) => input.toString() === '/api/games/geopolitix/draft')).toBe(true))

    let scoreInput = await screen.findByLabelText('Total score') as HTMLInputElement
    expect(scoreInput.value).toBe('321')
    await user.click(screen.getByRole('button', { name: 'Choose another' }))
    await user.upload(screen.getAllByLabelText('Result screenshot')[0], file)
    fireEvent.submit(screen.getAllByRole('button', { name: 'Analyze screenshot' })[0].closest('form')!)
    scoreInput = await screen.findByLabelText('Total score') as HTMLInputElement
    await user.clear(scoreInput)
    await user.type(scoreInput, '300')
    await user.click(screen.getByRole('button', { name: 'Confirm score' }))

    expect(await screen.findByText(/Submitted:/)).toBeTruthy()
    expect(screen.getByRole('link', { name: 'View screenshot' }).getAttribute('href')).toBe('/api/screenshots/draft-1')
  })

  it('shows owner loading before the empty state', async () => {
    const owner = { role: 'owner', name: 'Owner' }
    let resolveSubmissions!: (response: Response) => void
    const pendingSubmissions = new Promise<Response>((resolve) => { resolveSubmissions = resolve })
    vi.stubGlobal('fetch', vi.fn(async (input: string | URL | Request) => {
      const path = input.toString()
      if (path === '/api/session') return Response.json(owner)
      if (path === '/api/dashboard') return Response.json({ ...dashboard, viewer: owner })
      if (path === '/api/owner/submissions') return pendingSubmissions
      throw new Error(`unexpected request: ${path}`)
    }))
    render(<App />)

    expect((await screen.findByText('Loading owner controls…')).getAttribute('role')).toBe('status')
    expect(screen.queryByText('No submissions to correct today.')).toBeNull()
    resolveSubmissions(Response.json([]))
    expect(await screen.findByText('No submissions to correct today.')).toBeTruthy()
  })

  it('lets the owner correct and reopen today’s submissions', async () => {
    const owner = { role: 'owner', name: 'Owner' }
    const ownerDashboard = { ...dashboard, viewer: owner }
    const submissions = [{ id: 'score-1', playerName: 'Alice', gameName: 'Geopolitix', rawScore: 300, normalizedScore: 33.333, screenshotUrl: '/api/screenshots/score-1' }]
    const fetchMock = vi.fn(async (input: string | URL | Request, init?: RequestInit) => {
      const path = input.toString()
      if (path === '/api/session') return Response.json(owner)
      if (path === '/api/dashboard') return Response.json(ownerDashboard)
      if (path === '/api/owner/submissions') return Response.json(submissions)
      if (path === '/api/owner/submissions/score-1/score') {
        expect(JSON.parse(init?.body as string)).toEqual({ score: 450 })
        return Response.json({ id: 'score-1', score: 450, normalizedScore: 50 })
      }
      if (path === '/api/owner/submissions/score-1') return new Response(null, { status: 204 })
      throw new Error(`unexpected request: ${path}`)
    })
    vi.stubGlobal('fetch', fetchMock)
    vi.spyOn(window, 'confirm').mockReturnValue(true)
    const user = userEvent.setup()
    render(<App />)

    const input = await screen.findByLabelText('Raw score') as HTMLInputElement
    await user.clear(input)
    await user.type(input, '450')
    await user.click(screen.getByRole('button', { name: 'Save score' }))
    await waitFor(() => expect(fetchMock.mock.calls.some(([path]) => path === '/api/owner/submissions/score-1/score')).toBe(true))

    await user.click(screen.getByRole('button', { name: 'Reopen' }))
    await waitFor(() => expect(fetchMock).toHaveBeenCalledWith('/api/owner/submissions/score-1', { method: 'DELETE' }))
    expect(window.confirm).toHaveBeenCalled()
  })

  it('shows an invalid-token error without leaving the login screen', async () => {
    vi.stubGlobal('fetch', vi.fn(async (input: string | URL | Request) => {
      if (input.toString() === '/api/session') return new Response('', { status: 401 })
      return new Response('invalid token', { status: 401 })
    }))
    const user = userEvent.setup()
    render(<App />)

    await user.type(await screen.findByLabelText('Token'), 'wrong')
    await user.click(screen.getByRole('button', { name: 'Sign in' }))

    expect((await screen.findByRole('alert')).textContent).toBe('That token is not valid.')
    expect(screen.getByRole('heading', { name: 'Welcome back' })).toBeTruthy()
  })

  it('enables and disables browser push notifications', async () => {
    let active = false
    const subscription = {
      endpoint: 'https://push.example/subscription',
      toJSON: () => ({ endpoint: 'https://push.example/subscription', keys: { p256dh: 'key', auth: 'auth' } }),
      unsubscribe: vi.fn(async () => { active = false; return true }),
    }
    const registration = {
      pushManager: {
        getSubscription: vi.fn(async () => active ? subscription : null),
        subscribe: vi.fn(async () => { active = true; return subscription }),
      },
    }
    vi.stubGlobal('navigator', Object.assign(Object.create(navigator), {
      serviceWorker: { register: vi.fn(async () => registration), ready: Promise.resolve(registration), getRegistration: vi.fn(async () => registration) },
    }))
    vi.stubGlobal('PushManager', class {})
    vi.stubGlobal('Notification', { requestPermission: vi.fn(async () => 'granted') })
    const fetchMock = vi.fn(async (input: string | URL | Request, init?: RequestInit) => {
      const path = input.toString()
      if (path === '/api/session') return Response.json(player)
      if (path === '/api/dashboard') return Response.json(dashboard)
      if (path === '/api/push/config') return Response.json({ enabled: true, publicKey: 'AQID' })
      if (path === '/api/push/subscriptions' && init?.method === 'POST') return new Response(null, { status: 204 })
      if (path === '/api/push/subscriptions' && init?.method === 'DELETE') return new Response(null, { status: 204 })
      throw new Error(`unexpected request: ${path}`)
    })
    vi.stubGlobal('fetch', fetchMock)
    const user = userEvent.setup()
    render(<App />)

    await user.click(await screen.findByRole('button', { name: 'Enable notifications' }))
    await waitFor(() => expect(registration.pushManager.subscribe).toHaveBeenCalled())
    expect(fetchMock.mock.calls.some(([path, init]) => path === '/api/push/subscriptions' && init?.method === 'POST')).toBe(true)
    await user.click(screen.getByRole('button', { name: 'Disable notifications' }))
    await waitFor(() => expect(subscription.unsubscribe).toHaveBeenCalled())
    expect(fetchMock.mock.calls.some(([path, init]) => path === '/api/push/subscriptions' && init?.method === 'DELETE')).toBe(true)
  })

  it('refreshes the dashboard after a leaderboard event', async () => {
    let listener: (() => void) | undefined
    class FakeEventSource {
      addEventListener(_name: string, callback: EventListenerOrEventListenerObject) {
        listener = callback as () => void
      }
      removeEventListener() {}
      close() {}
    }
    vi.stubGlobal('EventSource', FakeEventSource)
    const fetchMock = vi.fn(async (input: string | URL | Request) => {
      if (input.toString() === '/api/session') return Response.json(player)
      return Response.json(dashboard)
    })
    vi.stubGlobal('fetch', fetchMock)
    render(<App />)

    await screen.findByRole('heading', { name: 'Scores' })
    listener?.()
    await waitFor(() => expect(fetchMock.mock.calls.filter(([input]) => input.toString() === '/api/dashboard')).toHaveLength(2))
  })

  it('recovers from a failed session check', async () => {
    vi.stubGlobal('fetch', vi.fn().mockRejectedValue(new Error('offline')))
    render(<App />)
    expect(await screen.findByRole('heading', { name: 'Welcome back' })).toBeTruthy()
  })

  it('reports dashboard and logout failures', async () => {
    const fetchMock = vi.fn(async (input: string | URL | Request) => {
      const path = input.toString()
      if (path === '/api/session') return Response.json(player)
      if (path === '/api/dashboard') return new Response('', { status: 500 })
      if (path === '/api/logout') throw new Error('offline')
      throw new Error(`unexpected request: ${path}`)
    })
    vi.stubGlobal('fetch', fetchMock)
    const user = userEvent.setup()
    render(<App />)

    expect((await screen.findByRole('alert')).textContent).toBe('Could not load today’s leaderboard.')
    await user.click(screen.getByRole('button', { name: 'Log out' }))
    expect((await screen.findByRole('alert')).textContent).toBe('Could not log out. Try again.')
    expect(screen.getByRole('heading', { name: 'Daily leaderboard' })).toBeTruthy()
  })

  it('restores an existing session and logs out', async () => {
    const fetchMock = vi.fn(async (input: string | URL | Request) => {
      const path = input.toString()
      if (path === '/api/session') return Response.json(player)
      if (path === '/api/dashboard') return Response.json(dashboard)
      if (path === '/api/logout') return new Response(null, { status: 204 })
      throw new Error(`unexpected request: ${path}`)
    })
    vi.stubGlobal('fetch', fetchMock)
    const user = userEvent.setup()
    render(<App />)

    await user.click(await screen.findByRole('button', { name: 'Log out' }))
    expect(await screen.findByRole('heading', { name: 'Welcome back' })).toBeTruthy()
    await waitFor(() => expect(fetchMock).toHaveBeenCalledWith('/api/logout', { method: 'POST' }))
  })
})
