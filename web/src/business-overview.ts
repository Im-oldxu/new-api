import { lazy, type ComponentType } from 'react'

export const businessOverviews: { id: string; component: ComponentType }[] = [
  { id: 'tickets', component: lazy(() => import('@/features/tickets/overview')) },
]
