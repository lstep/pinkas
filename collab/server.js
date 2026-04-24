const { Server } = require('@hocuspocus/server')
const Y = require('yjs')
const { defaultMarkdownSerializer } = require('prosemirror-markdown')
const { yXmlFragmentToProseMirrorRootNode } = require('y-prosemirror')
const http = require('http')

const API_URL = process.env.API_URL || 'http://localhost:3000'
const PORT = parseInt(process.env.PORT || '3001', 10)

// Internal API calls to Go backend
async function callInternal(path, options = {}) {
  const url = `${API_URL}${path}`
  const res = await fetch(url, {
    ...options,
    headers: { 'Content-Type': 'application/json', ...options.headers },
  })
  return { status: res.status, data: await res.json().catch(() => null) }
}

const server = new Server({
  async onLoadDocument({ documentName }) {
    console.log('[onLoadDocument] loading:', documentName)
    const { status, data: body } = await callInternal(
      `/internal/load?docId=${encodeURIComponent(documentName)}`
    )
    const ydoc = new Y.Doc()
    if (status === 200 && body?.yjsSnapshot) {
      try {
        const buffer = Buffer.from(body.yjsSnapshot, 'base64')
        Y.applyUpdate(ydoc, new Uint8Array(buffer))
        console.log('[onLoadDocument] loaded state, size:', buffer.length)
      } catch (err) {
        console.error('[onLoadDocument] failed to apply state:', err.message)
      }
    } else {
      console.log('[onLoadDocument] no snapshot found, starting empty')
    }
    return ydoc
  },

  async onAuthenticate(data) {
    const { token, documentName } = data
    const { status, data: body } = await callInternal(
      `/internal/auth?token=${encodeURIComponent(token)}&docId=${encodeURIComponent(documentName)}`
    )
    if (status !== 200 || !body?.userId) {
      console.error('[onAuthenticate] failed:', status, body, 'token length:', token?.length)
      throw new Error('Authentication failed')
    }
    console.log('[onAuthenticate] success:', body.userId, body.permission)
    return { userId: body.userId, permission: body.permission }
  },

  async onStoreDocument({ documentName, document }) {
    const snapshot = Buffer.from(Y.encodeStateAsUpdate(document))
    const markdown = yjsToMarkdown(document)

    await callInternal('/internal/save', {
      method: 'POST',
      body: JSON.stringify({
        docId: documentName,
        markdown,
        yjsSnapshot: snapshot.toString('base64'),
        authorId: 'system',
      }),
    })
  },

  async onDestroy(data) {
    await callInternal('/internal/cleanup', {
      method: 'POST',
      body: JSON.stringify({ docId: data.documentName }),
    })
  },
})

// Yjs → ProseMirror → Markdown conversion
function yjsToMarkdown(ydoc) {
  try {
    const yXmlFragment = ydoc.getXmlFragment('prosemirror')
    const pmNode = yXmlFragmentToProseMirrorRootNode(yXmlFragment)
    return defaultMarkdownSerializer.serialize(pmNode)
  } catch (err) {
    return ''
  }
}

// Health endpoint
const healthServer = http.createServer((req, res) => {
  if (req.url === '/health') {
    res.writeHead(200, { 'Content-Type': 'application/json' })
    res.end(JSON.stringify({ status: 'ok' }))
  } else {
    res.writeHead(404)
    res.end()
  }
})

// Start servers explicitly
async function start() {
  await server.listen(PORT)
  healthServer.listen(3002, () => {
    console.log(`Hocuspocus listening on :${PORT}, health on :3002`)
  })
}

start().catch(err => {
  console.error('Failed to start collab server:', err)
  process.exit(1)
})
