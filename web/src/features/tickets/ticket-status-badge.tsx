import { useTranslation } from 'react-i18next'

import { StatusBadge } from '@/components/status-badge'

import { states, type TicketState } from './api'

export function TicketStatusBadge({ status }: { status: TicketState }) {
  const { t } = useTranslation()
  const { label, tone } = states[status]
  const text = t(label)

  return (
    <StatusBadge
      label={text}
      variant={tone}
      type='badge'
      copyable={false}
      showDot
      className='border border-current/35 bg-current/10'
    >
      <span className='truncate leading-normal text-[color:color-mix(in_oklab,currentColor_40%,var(--foreground))]'>
        {text}
      </span>
    </StatusBadge>
  )
}
