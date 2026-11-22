import type { TFunction } from 'i18next'
import { Ticket } from 'lucide-react'
import type { NavGroup } from '@/components/layout/types'

export function ticketNavigation(t: TFunction): NavGroup {
  return { id: 'work', title: t('Workspace'), items: [{ title: t('Tickets'), url: '/tickets', icon: Ticket }] }
}
