export type Viewer = {
  role: 'player' | 'owner'
  playerId?: string
  name: string
}

export type Game = {
  id: string
  name: string
  url: string
  maxScore: number
}

export type OwnerSubmission = {
  id: string
  playerName: string
  gameName: string
  rawScore: number
  normalizedScore: number
  screenshotUrl: string
}
