import http from 'node:http'

const server = http.createServer((request, response) => {
  if (request.method === 'GET') {
    response.writeHead(200).end('ok')
    return
  }
  let body = ''
  request.on('data', (chunk) => { body += chunk })
  request.on('end', () => {
    const prompt = JSON.parse(body).messages[0].content[0].text
    const score = prompt.startsWith('Analyze this screenshot from Krillion.') ? 4500 : 321
    const content = JSON.stringify({ valid: true, score, reason: '' })
    response.setHeader('Content-Type', 'application/json')
    response.end(JSON.stringify({ choices: [{ message: { content } }] }))
  })
})

server.listen(18081, '127.0.0.1')
process.on('SIGTERM', () => server.close())
