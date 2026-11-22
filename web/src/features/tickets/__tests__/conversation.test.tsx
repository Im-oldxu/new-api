import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen } from '@testing-library/react'
import { expect, it } from 'vitest'

import type { TicketDetail } from '../api'
import { TicketConversation } from '../conversation'

const ticket: TicketDetail = {
  id: 7,
  owner_id: 1,
  owner_name: 'Ari',
  assignee_id: 2,
  assignee_name: 'Kim',
  title: 'Display cable',
  description: 'Replace cable',
  status: 'in_progress',
  attachment_name: '',
  version: 4,
  created_at: 10,
  updated_at: 20,
  entries: [
    {
      id: 9,
      action: 'reply',
      body: 'Cable ordered',
      actor_id: 2,
      actor_name: 'Kim',
      created_at: 20,
    },
  ],
}

it.each([
  ['pending', 'Awaiting processing', 'warning'],
  ['in_progress', 'In progress', 'info'],
  ['completed', 'Completed', 'success'],
] as const)(
  'displays %s as a read-only status with its semantic color',
  (status, label, tone) => {
    render(
      <QueryClientProvider client={new QueryClient()}>
        <TicketConversation
          ticket={{ ...ticket, status }}
          userId={1}
          role={1}
          all={false}
          canWrite={false}
          edit={() => {}}
          refresh={() => {}}
        />
      </QueryClientProvider>
    )
    const badge = screen.getByTitle(label)
    expect(badge).toHaveTextContent(label)
    expect(badge).toHaveClass(`text-${tone}`)
    expect(badge.closest('a,button,[role="button"]')).toBeNull()
  }
)

it.each([
  ['owner', 1, 1, false, false, true, false],
  ['assigned administrator', 2, 10, true, true, true, true],
  ['other administrator', 3, 10, true, true, false, false],
  ['read-only administrator', 2, 10, true, false, false, false],
  ['root', 4, 100, true, true, true, true],
] as const)(
  'shows permitted actions to the %s on an assigned ticket',
  (_, id, role, all, canWrite, reply, complete) => {
    render(
      <QueryClientProvider client={new QueryClient()}>
        <TicketConversation
          ticket={ticket}
          userId={id}
          role={role}
          all={all}
          canWrite={canWrite}
          edit={() => {}}
          refresh={() => {}}
        />
      </QueryClientProvider>
    )
    expect(screen.getByText('Cable ordered')).toBeVisible()
    expect(Boolean(screen.queryByRole('textbox', { name: 'Reply' }))).toBe(
      reply
    )
    expect(
      Boolean(screen.queryByRole('button', { name: 'Complete ticket' }))
    ).toBe(complete)
    expect(
      screen.queryByRole('button', { name: 'Claim ticket' })
    ).not.toBeInTheDocument()
  }
)
