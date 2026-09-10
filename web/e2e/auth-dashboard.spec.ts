import { expect, test } from '@playwright/test'

test('player signs in, restores the session, views games, and logs out', async ({ page }) => {
  await page.goto('/')

  await expect(page.getByRole('heading', { name: 'Welcome back' })).toBeVisible()
  await page.getByLabel('Token').fill('e2e-alice-token-123456')
  await page.getByRole('button', { name: 'Sign in' }).click()

  await expect(page.getByRole('heading', { name: 'Daily leaderboard' })).toBeVisible()
  await expect(page.getByText('Signed in as Alice')).toBeVisible()
  const playLinks = page.getByRole('link', { name: 'Play game →' })
  await expect(playLinks.nth(0)).toHaveAttribute('href', 'https://geopolitix.live/')
  await expect(playLinks.nth(1)).toHaveAttribute('href', 'https://krillion.io/')
  await expect(page.getByRole('row', { name: /Alice 0\/2/ })).toBeVisible()
  await expect(page.getByRole('row', { name: /Bob 0\/2/ })).toBeVisible()

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

test('submission updates another player live and shares its screenshot', async ({ page, browser }) => {
  await page.goto('/')
  await page.getByLabel('Token').fill('e2e-alice-token-123456')
  await page.getByRole('button', { name: 'Sign in' }).click()

  const bobContext = await browser.newContext()
  const bobPage = await bobContext.newPage()
  await bobPage.goto('/')
  const eventStream = bobPage.waitForResponse((response) => response.url().endsWith('/api/events'))
  await bobPage.getByLabel('Token').fill('e2e-bob-token-12345678')
  await bobPage.getByRole('button', { name: 'Sign in' }).click()
  await expect(bobPage.getByRole('heading', { name: 'Scores' })).toBeVisible()
  await eventStream

  const game = page.locator('article').filter({ hasText: 'Geopolitix' })
  await game.getByLabel('Result screenshot').setInputFiles({
    name: 'score.png',
    mimeType: 'image/png',
    buffer: Buffer.from('iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mNk+A8AAQUBAScY42YAAAAASUVORK5CYII=', 'base64'),
  })
  await game.getByRole('button', { name: 'Analyze screenshot' }).click()

  await expect(game.getByLabel('Total score')).toHaveValue('321')
  await game.getByLabel('Total score').fill('300')
  await game.getByRole('button', { name: 'Confirm score' }).click()

  await expect(game.getByText(/Submitted:/)).toContainText('300')
  const screenshotURL = await game.getByRole('link', { name: 'View screenshot' }).getAttribute('href')
  await expect(page.getByRole('row', { name: /#1 Alice 1\/2 300/ })).toBeVisible()
  await expect(bobPage.getByRole('row', { name: /#1 Alice 1\/2 300/ })).toBeVisible()

  const response = await bobContext.request.get(`http://127.0.0.1:18080${screenshotURL}`)
  expect(response.status()).toBe(200)
  expect(response.headers()['content-type']).toBe('image/png')

  const ownerContext = await browser.newContext()
  const ownerPage = await ownerContext.newPage()
  await ownerPage.goto('/')
  await ownerPage.getByLabel('Token').fill('e2e-owner-token-123456')
  await ownerPage.getByRole('button', { name: 'Sign in' }).click()
  const scoreInput = ownerPage.getByLabel('Raw score')
  await expect(scoreInput).toHaveValue('300')
  await scoreInput.fill('450')
  await ownerPage.getByRole('button', { name: 'Save score' }).click()
  await expect(bobPage.getByRole('row', { name: /#1 Alice 1\/2 450/ })).toBeVisible()

  ownerPage.once('dialog', (dialog) => dialog.accept())
  await ownerPage.getByRole('button', { name: 'Reopen' }).click()
  await expect(page.locator('article').filter({ hasText: 'Geopolitix' }).getByLabel('Result screenshot')).toBeVisible()
  await expect(bobPage.getByRole('row', { name: /#1 Alice 0\/2/ })).toBeVisible()

  await ownerContext.close()
  await bobContext.close()
})
