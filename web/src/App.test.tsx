// @vitest-environment jsdom

import { cleanup, render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { App } from './App'

const player = { role: 'player', playerId: 'alice', name: 'Alice' }
const dashboard = {
  date: '2026-09-10',
  viewer: player,
  games: [
    { id: 'geopolitix', name: 'Geopolitix', url: 'https://geopolitix.live/' },
    { id: 'krillion', name: 'Krillion', url: 'https://krillion.io/' },
  ],
  players: [
    { id: 'alice', name: 'Alice', completed: 0, combinedScore: 0 },
    { id: 'bob', name: 'Bob', completed: 0, combinedScore: 0 },
  ],
}

afterEach(() => {
  cleanup()
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
    expect(screen.getByRole('link', { name: /Geopolitix/ }).getAttribute('href')).toBe('https://geopolitix.live/')
    expect(screen.getByText('Alice')).toBeTruthy()
    expect(screen.getByText('Bob')).toBeTruthy()
    expect(screen.getAllByText('0/2')).toHaveLength(2)
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
