import { useQuery, useQueryClient } from '@tanstack/react-query'
import {
  useReactTable,
  getCoreRowModel,
  type ColumnDef,
} from '@tanstack/react-table'
import { Plus } from 'lucide-react'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'

import { DataTablePage } from '@/components/data-table'
import { Dialog } from '@/components/dialog'
import { SectionPageLayout } from '@/components/layout'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { NativeSelect, NativeSelectOption } from '@/components/ui/native-select'
import { Tabs, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { hasPermission } from '@/lib/admin-permissions'
import { formatTimestamp } from '@/lib/format'
import { useAuthStore } from '@/stores/auth-store'

import {
  getTicketSettings,
  getTicket,
  listTickets,
  states,
  type Ticket,
} from './api'
import { TicketConversation } from './conversation'
import { TicketEditor } from './editor'
import { TicketSettingsDialog } from './settings'
import { TicketStatusBadge } from './ticket-status-badge'

export function TicketsPage({
  initial,
}: {
  initial: {
    scope?: string
    status?: string
    assignment?: string
    id?: number
    create?: boolean
  }
}) {
  const { t } = useTranslation()
  const client = useQueryClient()
  const user = useAuthStore((s) => s.auth.user)
  const [scope, setScope] = useState(initial.scope ?? 'self')
  const all = scope === 'all' && hasPermission(user, 'ticket', 'read')
  const [search, setSearch] = useState('')
  const [status, setStatus] = useState(initial.status ?? '')
  const [assignment, setAssignment] = useState(initial.assignment ?? '')
  const [showSettings, setShowSettings] = useState(false)
  const settings = useQuery({
    queryKey: ['ticket-settings'],
    queryFn: getTicketSettings,
  })
  const [pagination, setPagination] = useState({ pageIndex: 0, pageSize: 10 })
  const [active, setActive] = useState<number | null>(initial.id ?? null)
  const [editor, setEditor] = useState<Ticket | 'new' | null>(
    initial.create ? 'new' : null
  )
  const list = useQuery({
    queryKey: [
      'tickets',
      user?.id,
      all,
      status,
      assignment,
      search,
      pagination,
    ],
    queryFn: () =>
      listTickets(all, {
        p: pagination.pageIndex + 1,
        page_size: pagination.pageSize,
        status,
        search,
        assignment,
      }),
  })
  const detail = useQuery({
    refetchInterval: active === null ? false : 30_000,
    queryKey: ['ticket', user?.id, all, active],
    enabled: active !== null,
    queryFn: () => getTicket(all, Number(active)),
  })
  function refresh() {
    void client.invalidateQueries({ queryKey: ['ticket-overview'] })
    void client.invalidateQueries({ queryKey: ['tickets', user?.id] })
    void client.invalidateQueries({ queryKey: ['ticket', user?.id] })
  }
  const columns: ColumnDef<Ticket>[] = [
    { accessorKey: 'id', header: 'ID' },
    {
      accessorKey: 'title',
      header: t('Title'),
      cell: ({ row }) => (
        <Button
          variant='link'
          className='max-w-full truncate px-0'
          onClick={() => setActive(row.original.id)}
        >
          {row.original.title}
        </Button>
      ),
    },
    {
      accessorKey: 'status',
      header: t('Status'),
      cell: ({ row }) => <TicketStatusBadge status={row.original.status} />,
    },
    {
      accessorKey: 'owner_name',
      header: t('Submitted by'),
      cell: ({ row }) => row.original.owner_name || `#${row.original.owner_id}`,
    },
    {
      accessorKey: 'assignee_name',
      header: t('Assigned to'),
      cell: ({ row }) =>
        row.original.assignee_name ||
        (row.original.assignee_id
          ? `#${row.original.assignee_id}`
          : t('None')),
    },
    {
      accessorKey: 'updated_at',
      header: t('Updated'),
      cell: ({ row }) => formatTimestamp(row.original.updated_at),
    },
    {
      id: 'actions',
      header: t('Actions'),
      cell: ({ row }) => (
        <Button
          variant='outline'
          size='sm'
          onClick={() => setActive(row.original.id)}
        >
          {t('Details')}
        </Button>
      ),
    },
  ]
  const table = useReactTable({
    data: list.data?.items ?? [],
    columns,
    state: { pagination },
    onPaginationChange: setPagination,
    getCoreRowModel: getCoreRowModel(),
    manualPagination: true,
    rowCount: list.data?.total ?? 0,
    getRowId: (r) => String(r.id),
  })
  const filtered = Boolean(search.trim() || status || assignment)
  let emptyTitle = t('No tickets yet')
  let emptyAction = (
    <Button onClick={() => setEditor('new')}>{t('Create ticket')}</Button>
  )
  if (filtered) {
    emptyTitle = t('No matching results')
    emptyAction = (
      <Button
        variant='outline'
        onClick={() => {
          setAssignment('')
          setSearch('')
          setStatus('')
          setPagination({ ...pagination, pageIndex: 0 })
        }}
      >
        {t('Clear filters')}
      </Button>
    )
  }
  if (list.isError) {
    emptyTitle = t('Failed to load')
    emptyAction = (
      <Button onClick={() => void list.refetch()}>{t('Retry')}</Button>
    )
  }
  const row = detail.data
  return (
    <SectionPageLayout fixedContent>
      <SectionPageLayout.Title>{t('Tickets')}</SectionPageLayout.Title>
      <SectionPageLayout.Actions>
        {user?.role === 100 && (
          <Button variant='outline' onClick={() => setShowSettings(true)}>
            {t('Ticket settings')}
          </Button>
        )}
        <Button onClick={() => setEditor('new')}>
          <Plus />
          {t('Create ticket')}
        </Button>
      </SectionPageLayout.Actions>
      <SectionPageLayout.Content>
        <DataTablePage
          table={table}
          columns={columns}
          isLoading={list.isPending}
          emptyTitle={emptyTitle}
          emptyAction={emptyAction}
          hideMobile
          paginationInFooter
          toolbar={
            <div className='flex flex-wrap items-center gap-2'>
              {hasPermission(user, 'ticket', 'read') && (
                <Tabs
                  value={scope}
                  onValueChange={(v) => {
                    setActive(null)
                    setAssignment('')
                    setScope(v)
                    setPagination({ ...pagination, pageIndex: 0 })
                  }}
                >
                  <TabsList>
                    <TabsTrigger value='self'>{t('Only Mine')}</TabsTrigger>
                    <TabsTrigger value='all'>{t('All tickets')}</TabsTrigger>
                  </TabsList>
                </Tabs>
              )}
              <Input
                aria-label={t('Search')}
                placeholder={t('Search')}
                className='max-w-64'
                value={search}
                onChange={(e) => {
                  setSearch(e.target.value)
                  setPagination({ ...pagination, pageIndex: 0 })
                }}
              />
              {all && (
                <NativeSelect
                  aria-label={t('Assigned to')}
                  value={assignment}
                  onChange={(e) => {
                    setAssignment(e.target.value)
                    setPagination({ ...pagination, pageIndex: 0 })
                  }}
                >
                  <NativeSelectOption value=''>{t('All')}</NativeSelectOption>
                  <NativeSelectOption value='unassigned'>
                    {t('Unassigned')}
                  </NativeSelectOption>
                  <NativeSelectOption value='mine'>
                    {t('Assigned to me')}
                  </NativeSelectOption>
                </NativeSelect>
              )}
              <NativeSelect
                aria-label={t('Status')}
                value={status}
                onChange={(e) => {
                  setStatus(e.target.value)
                  setPagination({ ...pagination, pageIndex: 0 })
                }}
              >
                <NativeSelectOption value=''>
                  {t('All statuses')}
                </NativeSelectOption>
                {Object.entries(states).map(([s, { label }]) => (
                  <NativeSelectOption key={s} value={s}>
                    {t(label)}
                  </NativeSelectOption>
                ))}
              </NativeSelect>
            </div>
          }
        />
        {active !== null && !editor && (
          <Dialog
            open
            onOpenChange={(open) => {
              if (!open) setActive(null)
            }}
            title={row?.title ?? t('Details')}
            titleClassName='pr-5 wrap-anywhere'
          >
            {row ? (
              <TicketConversation
                key={row.id}
                ticket={row}
                all={all}
                userId={user?.id}
                role={user?.role}
                canWrite={hasPermission(user, 'ticket', 'write')}
                edit={() => setEditor(row)}
                refresh={refresh}
              />
            ) : (
              <div>
                <p>{detail.isError ? t('Failed to load') : t('Loading...')}</p>
                {detail.isError && (
                  <Button onClick={() => void detail.refetch()}>
                    {t('Retry')}
                  </Button>
                )}
              </div>
            )}
          </Dialog>
        )}
        {(editor || showSettings) && !settings.data && (
          <Dialog
            open
            onOpenChange={() => {
              setEditor(null)
              setShowSettings(false)
            }}
            title={t('Ticket settings')}
          >
            {settings.isError ? (
              <Button onClick={() => void settings.refetch()}>
                {t('Retry')}
              </Button>
            ) : (
              t('Loading...')
            )}
          </Dialog>
        )}
        {showSettings && settings.data && (
          <TicketSettingsDialog
            limit={settings.data.attachment_limit_kib}
            close={() => setShowSettings(false)}
          />
        )}
        {editor && settings.data && (
          <TicketEditor
            limit={settings.data.attachment_limit_kib}
            ticket={editor === 'new' ? undefined : editor}
            close={() => setEditor(null)}
            saved={() => {
              setEditor(null)
              refresh()
            }}
          />
        )}
      </SectionPageLayout.Content>
    </SectionPageLayout>
  )
}
