import { createFileRoute } from '@tanstack/react-router'
import { z } from 'zod'

import { TicketsPage } from '@/features/tickets'

export const Route = createFileRoute('/_authenticated/tickets/')({
  validateSearch: z.object({
    scope: z.enum(['self', 'all']).optional().catch(undefined),
    status: z
      .enum(['pending', 'in_progress', 'completed'])
      .optional()
      .catch(undefined),
    assignment: z.enum(['unassigned', 'mine']).optional().catch(undefined),
    id: z.number().int().positive().optional().catch(undefined),
    create: z.boolean().optional().catch(undefined),
  }),
  component: TicketRoute,
})

function TicketRoute() {
  const initial = Route.useSearch()
  return <TicketsPage key={JSON.stringify(initial)} initial={initial} />
}
