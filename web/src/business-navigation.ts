import type { TFunction } from 'i18next'
import type { NavGroup } from '@/components/layout/types'
import { ticketNavigation } from '@/features/tickets/navigation'

const modules: ((t: TFunction) => NavGroup)[] = [
  ticketNavigation,
]

export function businessNavigation(t: TFunction): NavGroup[] {
  return modules.map((navigation) => navigation(t))
}
