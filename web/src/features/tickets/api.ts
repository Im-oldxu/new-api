import { api } from '@/lib/api'
import { requireServerSuccess } from '@/lib/server-error-message'

export const states = {
  pending: { label: 'Awaiting processing', tone: 'warning' },
  in_progress: { label: 'In progress', tone: 'info' },
  completed: { label: 'Completed', tone: 'success' },
} as const
export type TicketState = keyof typeof states
export interface Ticket {
  id: number
  owner_name: string
  assignee_name: string
  assignee_id: number | null
  owner_id: number
  title: string
  description: string
  status: TicketState
  version: number
  attachment_name: string
  created_at: number
  updated_at: number
}

export interface TicketDetail extends Ticket {
  entries: {
    id: number
    action: 'created' | 'edited' | 'claimed' | 'completed' | 'reply'
    actor_id: number
    actor_name: string
    created_at: number
    body: string
  }[]
}

interface Result<T> {
  success: boolean
  message: string
  data: T
}
export function ticketsPath(all: boolean) {
  return all ? '/api/manage/tickets' : '/api/tickets'
}
export async function listTickets(
  all: boolean,
  params: {
    p: number
    page_size: number
    status: string
    search: string
    assignment?: string
  }
) {
  const res = await api.get<Result<{ items: Ticket[]; total: number }>>(
    ticketsPath(all),
    { params }
  )
  return requireServerSuccess(res.data).data
}
export async function getTicket(all: boolean, id: number) {
  const res = await api.get<Result<TicketDetail>>(`${ticketsPath(all)}/${id}`)
  return requireServerSuccess(res.data).data
}
export async function saveTicket(data: FormData, id?: number) {
  const result = id
    ? await api.put(`/api/tickets/${id}`, data)
    : await api.post('/api/tickets', data)
  requireServerSuccess(result.data)
}
export async function advanceTicket(
  ticket: Ticket,
  status: 'in_progress' | 'completed'
) {
  const result = await api.patch(`${ticketsPath(true)}/${ticket.id}/status`, {
    status,
    version: ticket.version,
  })
  requireServerSuccess(result.data)
}
export async function replyToTicket(
  all: boolean,
  ticket: Ticket,
  body: string
) {
  const result = await api.post(`${ticketsPath(all)}/${ticket.id}/replies`, {
    version: ticket.version,
    body,
  })
  requireServerSuccess(result.data)
}

export async function downloadTicket(all: boolean, ticket: Ticket) {
  const res = await api.get<Blob>(
    `${ticketsPath(all)}/${ticket.id}/attachment`,
    { responseType: 'blob' }
  )
  const url = URL.createObjectURL(res.data)
  const link = document.createElement('a')
  link.href = url
  link.download = ticket.attachment_name
  link.click()
  window.setTimeout(() => URL.revokeObjectURL(url), 1000)
}

interface TicketOverviewData {
  total: number
  counts: Record<TicketState, number>
  start: number
  trend: number[]
  recent: Ticket[]
}
export async function getTicketOverview(all: boolean, days: number) {
  const res = await api.get<Result<TicketOverviewData>>(
    `${ticketsPath(all)}/overview`,
    { params: { days } }
  )
  return requireServerSuccess(res.data).data
}
export async function getTicketSettings() {
  return requireServerSuccess(
    (
      await api.get<Result<{ attachment_limit_kib: number }>>(
        '/api/tickets/settings'
      )
    ).data
  ).data
}
export async function updateTicketSettings(limit: number) {
  return requireServerSuccess(
    (
      await api.put<Result<{ attachment_limit_kib: number }>>(
        '/api/tickets/settings',
        { attachment_limit_kib: limit }
      )
    ).data
  ).data
}
