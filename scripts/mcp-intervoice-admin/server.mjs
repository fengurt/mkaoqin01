/**
 * MCP stdio server: proxies Intervoice gateway admin data APIs for agents.
 * Uses stderr for diagnostics only (stdout is MCP protocol).
 *
 * Env:
 *   INTERVOICE_GATEWAY_URL — default http://127.0.0.1:8010
 *   INTERVOICE_ADMIN_TOKEN — JWT for an admin user (with or without "Bearer " prefix)
 */
import process from 'node:process'
import { McpServer } from '@modelcontextprotocol/sdk/server/mcp.js'
import { StdioServerTransport } from '@modelcontextprotocol/sdk/server/stdio.js'
import { z } from 'zod'

const baseURL = (process.env.INTERVOICE_GATEWAY_URL || 'http://127.0.0.1:8010').replace(/\/$/, '')
let bearerToken = process.env.INTERVOICE_ADMIN_TOKEN || ''
if (bearerToken.startsWith('Bearer ')) {
  bearerToken = bearerToken.slice(7)
}

function authHeaders() {
  if (!bearerToken) {
    throw new Error('Set INTERVOICE_ADMIN_TOKEN to an admin JWT from POST /v1/auth/login')
  }
  return {
    Authorization: `Bearer ${bearerToken}`,
    Accept: 'application/json',
    'Content-Type': 'application/json',
  }
}

async function httpJSON(method, path, bodyObj) {
  const url = `${baseURL}${path}`
  const opts = { method, headers: authHeaders() }
  if (bodyObj !== undefined && method !== 'GET') {
    opts.body = JSON.stringify(bodyObj)
  }
  const res = await fetch(url, opts)
  const text = await res.text()
  let parsed
  try {
    parsed = text ? JSON.parse(text) : {}
  } catch {
    parsed = { raw: text }
  }
  if (!res.ok) {
    const err = new Error(`HTTP ${res.status}: ${text.slice(0, 800)}`)
    err.status = res.status
    err.body = parsed
    throw err
  }
  return parsed
}

function textResult(obj) {
  const text = typeof obj === 'string' ? obj : JSON.stringify(obj, null, 2)
  return { content: [{ type: 'text', text }] }
}

const mcp = new McpServer(
  { name: 'intervoice-admin-data', version: '1.0.0' },
  {
    instructions:
      'Calls Intervoice /v1/admin/data/* via gateway. Requires admin JWT. Prefer syncUid + ifMatchUpdatedAt for two-way sync.',
  },
)

mcp.registerTool(
  'intervoice_admin_meta',
  { description: 'GET /v1/admin/data/meta — row counts and sync metadata.' },
  async () => textResult(await httpJSON('GET', '/v1/admin/data/meta')),
)

mcp.registerTool(
  'intervoice_admin_schema',
  { description: 'GET /v1/admin/data/schema — table/column reference for agents.' },
  async () => textResult(await httpJSON('GET', '/v1/admin/data/schema')),
)

const paginationSchema = {
  limit: z.number().int().min(1).max(500).optional(),
  offset: z.number().int().min(0).optional(),
}

mcp.registerTool(
  'intervoice_admin_list_users',
  {
    description: 'GET /v1/admin/data/users — paginated users (no passwords).',
    inputSchema: z.object(paginationSchema),
  },
  async (args) => {
    const q = new URLSearchParams()
    if (args.limit != null) q.set('limit', String(args.limit))
    if (args.offset != null) q.set('offset', String(args.offset))
    const qs = q.toString()
    const path = `/v1/admin/data/users${qs ? `?${qs}` : ''}`
    return textResult(await httpJSON('GET', path))
  },
)

mcp.registerTool(
  'intervoice_admin_list_attendance',
  {
    description: 'GET /v1/admin/data/attendance — filter by userId and/or updatedSince (ISO-like string).',
    inputSchema: z.object({
      ...paginationSchema,
      userId: z.number().int().optional(),
      updatedSince: z.string().optional(),
    }),
  },
  async (args) => {
    const q = new URLSearchParams()
    if (args.limit != null) q.set('limit', String(args.limit))
    if (args.offset != null) q.set('offset', String(args.offset))
    if (args.userId != null) q.set('userId', String(args.userId))
    if (args.updatedSince) q.set('updatedSince', args.updatedSince)
    const qs = q.toString()
    const path = `/v1/admin/data/attendance${qs ? `?${qs}` : ''}`
    return textResult(await httpJSON('GET', path))
  },
)

const upsertAttendanceSchema = z.object({
  syncUid: z.string().optional(),
  id: z.number().int().optional(),
  userId: z.number().int(),
  status: z.string(),
  location: z.string().optional(),
  reason: z.string().optional(),
  occurredAt: z.string().optional(),
  attachmentUrl: z.string().optional(),
  ifMatchUpdatedAt: z.string().optional(),
})

mcp.registerTool(
  'intervoice_admin_upsert_attendance',
  {
    description:
      'POST /v1/admin/data/attendance/upsert — create/update by syncUid or legacy id. Use ifMatchUpdatedAt to detect conflicts (409).',
    inputSchema: upsertAttendanceSchema,
  },
  async (body) => textResult(await httpJSON('POST', '/v1/admin/data/attendance/upsert', body)),
)

const patchUserSchema = z.object({
  id: z.number().int().optional(),
  syncUid: z.string().optional(),
  displayName: z.string().optional(),
  role: z.enum(['admin', 'employee']).optional(),
  ifMatchUpdatedAt: z.string().optional(),
})

mcp.registerTool(
  'intervoice_admin_patch_user',
  {
    description:
      'POST /v1/admin/data/users/patch — update displayName and/or role; assigns sync_uid if missing. Not for passwords.',
    inputSchema: patchUserSchema,
  },
  async (body) => textResult(await httpJSON('POST', '/v1/admin/data/users/patch', body)),
)

const transport = new StdioServerTransport()
if (!bearerToken) {
  console.error('mcp-intervoice-admin: INTERVOICE_ADMIN_TOKEN is not set; tool calls will fail until it is set.')
}
await mcp.connect(transport)
