import { useQuery } from '@tanstack/react-query'
import { Link } from '@tanstack/react-router'
import { ClipboardList } from 'lucide-react'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import {
  Bar,
  BarChart,
  CartesianGrid,
  Pie,
  PieChart,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from 'recharts'

import { Button } from '@/components/ui/button'
import { NativeSelect, NativeSelectOption } from '@/components/ui/native-select'
import { StatCard } from '@/features/dashboard/components/ui/stat-card'
import { hasPermission } from '@/lib/admin-permissions'
import { formatNumber, formatTimestamp } from '@/lib/format'
import { useAuthStore } from '@/stores/auth-store'

import { getTicketOverview, states, type TicketState } from './api'
import { TicketStatusBadge } from './ticket-status-badge'

const statuses = Object.keys(states) as TicketState[]

export default function TicketOverview() {
  const { t } = useTranslation()
  const user = useAuthStore((s) => s.auth.user)
  const canReadAll =
    !!user && user.role >= 10 && hasPermission(user, 'ticket', 'read')
  const [showAll, setShowAll] = useState(true)
  const all = canReadAll && showAll
  const scope = all ? 'all' : 'self'
  const [days, setDays] = useState(7)
  const result = useQuery({
    refetchInterval: 30_000,
    queryKey: ['ticket-overview', user?.id, scope, days],
    queryFn: () => getTicketOverview(all, days),
    enabled: !!user,
  })
  if (result.isPending) return <p role='status'>{t('Loading...')}</p>
  if (result.isError) {
    return (
      <div role='alert'>
        {t('Failed to load')}{' '}
        <Button variant='outline' onClick={() => void result.refetch()}>
          {t('Retry')}
        </Button>
      </div>
    )
  }
  const data = result.data
  const trend = data.trend.map((count, index) => ({
    day: new Date((data.start + index * 86400) * 1000)
      .toISOString()
      .slice(0, 10),
    count,
  }))
  const distribution = statuses.map((status) => ({
    status,
    label: t(states[status].label),
    fill: `var(--${states[status].tone})`,
    count: data.counts[status],
  }))
  return (
    <section className='space-y-5' aria-label={t('Tickets')}>
      <div className='flex flex-wrap items-center gap-2'>
        <h2 className='mr-auto text-lg font-semibold'>
          {canReadAll ? t('Tickets') : t('My tickets')}
        </h2>
        {canReadAll && (
          <NativeSelect
            aria-label={t('Scope')}
            value={scope}
            onChange={(event) => setShowAll(event.target.value === 'all')}
          >
            <NativeSelectOption value='self'>
              {t('My tickets')}
            </NativeSelectOption>
            <NativeSelectOption value='all'>
              {t('All tickets')}
            </NativeSelectOption>
          </NativeSelect>
        )}
        <Button render={<Link to='/tickets' search={{ create: true }} />}>
          {t('Create ticket')}
        </Button>
      </div>
      <div className='grid gap-3 sm:grid-cols-2 xl:grid-cols-4'>
        {(['total', ...statuses] as const).map((state) => (
          <Link
            key={state}
            to='/tickets'
            search={{ scope, status: state === 'total' ? undefined : state }}
          >
            <StatCard
              icon={ClipboardList}
              title={state === 'total' ? t('Total') : t(states[state].label)}
              iconTone={state === 'total' ? undefined : states[state].tone}
              value={formatNumber(
                state === 'total' ? data.total : data.counts[state]
              )}
              description={t('All time')}
            />
          </Link>
        ))}
      </div>
      {all && (
        <div className='flex flex-wrap gap-2'>
          <Button
            variant='outline'
            render={
              <Link
                to='/tickets'
                search={{ scope, assignment: 'unassigned' }}
              />
            }
          >
            {t('Unassigned')}
          </Button>
          <Button
            variant='outline'
            render={
              <Link to='/tickets' search={{ scope, assignment: 'mine' }} />
            }
          >
            {t('Assigned to me')}
          </Button>
        </div>
      )}
      <div className='grid gap-4 lg:grid-cols-2'>
        <section className='bg-card min-w-0 rounded-xl border p-4'>
          <div className='mb-3 flex flex-wrap items-center justify-between gap-2'>
            <h3 className='font-medium'>{t('New tickets (UTC)')}</h3>
            <NativeSelect
              aria-label={t('Time range')}
              value={days}
              onChange={(e) => setDays(Number(e.target.value))}
            >
              {[7, 30].map((n) => (
                <NativeSelectOption key={n} value={n}>
                  {t('Last {{days}} days', { days: n })}
                </NativeSelectOption>
              ))}
            </NativeSelect>
          </div>
          <ResponsiveContainer width='100%' height={180}>
            <BarChart data={trend}>
              <CartesianGrid vertical={false} />
              <XAxis
                dataKey='day'
                tickFormatter={(s) => String(s).slice(5)}
                minTickGap={16}
              />
              <YAxis allowDecimals={false} width={32} />
              <Tooltip />
              <Bar dataKey='count' name={t('Created')} fill='var(--chart-1)' />
            </BarChart>
          </ResponsiveContainer>
        </section>
        <section className='bg-card min-w-0 rounded-xl border p-4'>
          <h3 className='font-medium'>{t('Status distribution')}</h3>
          {data.total ? (
            <div className='flex flex-wrap items-center gap-3'>
              <PieChart width={170} height={165}>
                <Pie
                  data={distribution}
                  dataKey='count'
                  nameKey='label'
                  innerRadius={40}
                  outerRadius={65}
                />
                <Tooltip />
              </PieChart>
              <ul className='min-w-0 space-y-1'>
                {distribution.map((item) => (
                  <li key={item.status} className='flex items-center gap-2'>
                    <TicketStatusBadge status={item.status} />
                    <span className='tabular-nums'>
                      {formatNumber(item.count)}
                    </span>
                  </li>
                ))}
              </ul>
            </div>
          ) : (
            <p>{t('No tickets yet')}</p>
          )}
        </section>
      </div>
      <section className='bg-card rounded-xl border p-4'>
        <h3 className='mb-2 font-medium'>{t('Recently updated tickets')}</h3>
        {data.recent.length === 0 && <p>{t('No tickets yet')}</p>}
        <ul className='divide-y'>
          {data.recent.map((row) => (
            <li
              key={row.id}
              className='flex flex-wrap items-center justify-between gap-2 py-2'
            >
              <Link
                className='min-w-0 break-all hover:underline'
                to='/tickets'
                search={{ scope, id: row.id }}
              >
                #{row.id} · {row.title}
              </Link>
              <div className='flex min-w-0 flex-wrap items-center gap-2'>
                <TicketStatusBadge status={row.status} />
                <span className='text-muted-foreground text-sm'>
                  {formatTimestamp(row.updated_at)}
                </span>
              </div>
            </li>
          ))}
        </ul>
      </section>
    </section>
  )
}
