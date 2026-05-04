const { Server } = require('@hocuspocus/server')
const Y = require('yjs')
const { Schema } = require('prosemirror-model')
const { defaultMarkdownSerializer } = require('prosemirror-markdown')
const { yXmlFragmentToProseMirrorRootNode } = require('y-prosemirror')
const http = require('http')

// Define a schema matching Tiptap's default nodes for markdown serialization.
// This is needed because y-prosemirror's yXmlFragmentToProseMirrorRootNode
// requires a schema argument (no default in y-prosemirror 1.3.x).
const tiptapSchema = new Schema({
  nodes: {
    doc: { content: 'block+' },
    paragraph: { content: 'inline*', group: 'block' },
    text: { group: 'inline' },
    heading: { content: 'inline*', group: 'block', attrs: { level: { default: 1 } } },
    code_block: { content: 'text*', group: 'block' },
    hard_break: { inline: true, group: 'inline' },
    horizontal_rule: { group: 'block' },
    image: { group: 'inline', inline: true, attrs: { src: {}, alt: { default: null }, title: { default: null } } },
    ordered_list: { content: 'list_item+', group: 'block', attrs: { order: { default: 1 } } },
    bullet_list: { content: 'list_item+', group: 'block' },
    list_item: { content: 'paragraph block*' },
    blockquote: { content: 'block+', group: 'block' },
  },
  marks: {
    em: {},
    strong: {},
    link: { attrs: { href: {} } },
    code: {},
    strike: {},
    underline: {},
  }
})

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
  gc: false,
  debounce: 5000,

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

// Set a 5-minute auto-save interval
const AUTO_SAVE_INTERVAL = 5 * 60 * 1000 // 5 minutes

setInterval(async () => {
  try {
    const docs = server.hocuspocus?.documents
    if (docs && docs.size > 0) {
      console.log(`[auto-save] saving ${docs.size} active documents`)
      for (const [documentName, ydoc] of docs) {
        try {
          const snapshot = Buffer.from(Y.encodeStateAsUpdate(ydoc))
          const markdown = yjsToMarkdown(ydoc)

          await callInternal('/internal/save', {
            method: 'POST',
            body: JSON.stringify({
              docId: documentName,
              markdown,
              yjsSnapshot: snapshot.toString('base64'),
              authorId: 'system',
            }),
          })
          console.log(`[auto-save] saved: ${documentName}`)
        } catch (err) {
          console.error(`[auto-save] failed for ${documentName}:`, err.message)
        }
      }
    }
  } catch (err) {
    console.error('[auto-save] error:', err.message)
  }
}, AUTO_SAVE_INTERVAL)

// Yjs → ProseMirror → Markdown conversion
function yjsToMarkdown(ydoc) {
  const fragment = ydoc.getXmlFragment('prosemirror')
  
  // Log fragment content for debugging
  let childCount = 0
  try { fragment.forEach(() => childCount++) } catch (e) {}
  console.log('[yjsToMarkdown] fragment child count:', childCount)
  
  // Try primary path: y-prosemirror + prosemirror-markdown
  try {
    const pmNode = yXmlFragmentToProseMirrorRootNode(fragment, tiptapSchema)
    const md = defaultMarkdownSerializer.serialize(pmNode)
    if (md && md.trim()) {
      console.log('[yjsToMarkdown] success, length:', md.length)
      return md
    }
    console.log('[yjsToMarkdown] primary succeeded but result is empty')
  } catch (err) {
    console.error('[yjsToMarkdown] primary serialization failed:', err.message)
  }
  
  // Fallback: extract plain text directly from Yjs XML fragment
  try {
    return extractTextFromYFragment(fragment)
  } catch (fallbackErr) {
    console.error('[yjsToMarkdown] text fallback also failed:', fallbackErr.message)
    return ''
  }
}

// Fallback: extract plain text from a Yjs XML fragment
function extractTextFromYFragment(fragment) {
  const parts = []
  const walk = (node) => {
    // Y.XmlText nodes have toDelta(), Y.XmlElement has getAttribute()
    if (typeof node.toDelta === 'function') {
      // XmlText: extract text content
      try { parts.push(node.toString()) } catch (e) { /* skip unreadable nodes */ }
    } else if (typeof node.forEach === 'function') {
      // XmlFragment or XmlElement: recurse into children
      node.forEach(child => walk(child))
      // Add newline after block-level elements (not the root fragment)
      if (typeof node.getAttribute === 'function') parts.push('\n')
    }
  }
  walk(fragment)
  return parts.join('').trim()
}

// Health endpoint
const healthServer = http.createServer((req, res) => {
  // Set CORS headers for local communication
  res.setHeader('Content-Type', 'application/json')

  if (req.url === '/health' && req.method === 'GET') {
    res.writeHead(200)
    res.end(JSON.stringify({ status: 'ok' }))
  } else if (req.url === '/internal/restore' && req.method === 'POST') {
    let body = ''
    req.on('data', chunk => { body += chunk })
    req.on('end', async () => {
      try {
        const { docId, yjsSnapshot } = JSON.parse(body)
        if (!docId || !yjsSnapshot) {
          res.writeHead(400)
          res.end(JSON.stringify({ error: 'docId and yjsSnapshot required' }))
          return
        }

        // Get the managed document from Hocuspocus
        const hocuspocusDoc = server.documents.get(docId)
        if (!hocuspocusDoc) {
          res.writeHead(200)
          res.end(JSON.stringify({ status: 'ok', note: 'no active document' }))
          return
        }

        const ydoc = hocuspocusDoc.document

        // Create a new Y.Doc with the snapshot and apply its state
        const targetDoc = new Y.Doc()
        const buffer = Buffer.from(yjsSnapshot, 'base64')
        Y.applyUpdate(targetDoc, new Uint8Array(buffer))
        
        // Get the full state of the target document
        const targetUpdate = Y.encodeStateAsUpdate(targetDoc)

        // Apply to the managed document — this broadcasts to all connected clients
        Y.applyUpdate(ydoc, targetUpdate)

        // Save markdown to disk via Go API
        const markdown = yjsToMarkdown(ydoc)
        await callInternal('/internal/save', {
          method: 'POST',
          body: JSON.stringify({
            docId,
            markdown,
            yjsSnapshot,
            authorId: 'system',
          }),
        })

        console.log(`[restore] restored document: ${docId}`)
        res.writeHead(200)
        res.end(JSON.stringify({ status: 'ok' }))
      } catch (err) {
        console.error('[restore] error:', err.message)
        res.writeHead(500)
        res.end(JSON.stringify({ error: err.message }))
      }
    })
  } else {
    res.writeHead(404)
    res.end(JSON.stringify({ error: 'not found' }))
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
