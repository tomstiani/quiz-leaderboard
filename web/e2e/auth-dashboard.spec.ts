import { expect, test } from '@playwright/test'

test('player signs in, restores the session, views games, and logs out', async ({ page }) => {
  await page.goto('/')

  await expect(page.getByRole('heading', { name: 'Welcome back' })).toBeVisible()
  await page.getByLabel('Token').fill('e2e-alice-token-123456')
  await page.getByRole('button', { name: 'Sign in' }).click()

  await expect(page.getByRole('heading', { name: 'Daily leaderboard' })).toBeVisible()
  await expect(page.getByText('Signed in as Alice')).toBeVisible()
  await expect(page.getByRole('link', { name: /Geopolitix/ })).toHaveAttribute('href', 'https://geopolitix.live/')
  await expect(page.getByRole('link', { name: /Krillion/ })).toHaveAttribute('href', 'https://krillion.io/')
  await expect(page.getByRole('row', { name: /Alice 0\/2 0/ })).toBeVisible()
  await expect(page.getByRole('row', { name: /Bob 0\/2 0/ })).toBeVisible()

  await page.reload()
  await expect(page.getByText('Signed in as Alice')).toBeVisible()

  await page.getByRole('button', { name: 'Log out' }).click()
  await expect(page.getByRole('heading', { name: 'Welcome back' })).toBeVisible()
  await page.reload()
  await expect(page.getByRole('heading', { name: 'Welcome back' })).toBeVisible()
})

test('invalid token stays on the login screen', async ({ page }) => {
  await page.goto('/')
  await page.getByLabel('Token').fill('invalid-token')
  await page.getByRole('button', { name: 'Sign in' }).click()

  await expect(page.getByRole('alert')).toHaveText('That token is not valid.')
  await expect(page.getByRole('heading', { name: 'Welcome back' })).toBeVisible()
})

test('owner token creates an owner session', async ({ page }) => {
  await page.goto('/')
  await page.getByLabel('Token').fill('e2e-owner-token-123456')
  await page.getByRole('button', { name: 'Sign in' }).click()

  await expect(page.getByText('Signed in as Owner (owner)')).toBeVisible()
  await expect(page.getByRole('heading', { name: 'Scores' })).toBeVisible()
})
