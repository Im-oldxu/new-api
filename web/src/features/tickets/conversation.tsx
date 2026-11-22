import { useMutation } from '@tanstack/react-query'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { Button } from '@/components/ui/button'
import { Label } from '@/components/ui/label'
import { Textarea } from '@/components/ui/textarea'
import { formatTimestamp } from '@/lib/format'
import { handleServerError } from '@/lib/handle-server-error'
import { ROLE } from '@/lib/roles'

import {
  advanceTicket,
  downloadTicket,
  replyToTicket,
  type TicketDetail,
} from './api'
import { TicketStatusBadge } from './ticket-status-badge'

const entryLabels = {
  created: 'Created',
  edited: 'Edited',
  claimed: 'Claimed',
  reply: 'Reply',
  completed: 'Completed',
} as const

export function TicketConversation(props: {
  ticket: TicketDetail
  all: boolean
  userId?: number
  role?: number
  canWrite: boolean
  edit: () => void
  refresh: () => void
}) {
  const { t } = useTranslation()
  const [body, setBody] = useState('')
  const row = props.ticket
  const isOwner = row.owner_id === props.userId
  const open = row.status !== 'completed'
  const canManage = props.all && props.canWrite
  const canClaim = canManage && open && !row.assignee_id
  const handler =
    canManage &&
    !!row.assignee_id &&
    (row.assignee_id === props.userId || props.role === ROLE.SUPER_ADMIN)
  const canReply = open && (props.all ? handler : isOwner)
  const mutation = useMutation({
    mutationFn: (action: 'claim' | 'complete' | 'reply') => {
      if (action === 'reply') return replyToTicket(props.all, row, body.trim())
      return advanceTicket(
        row,
        action === 'claim' ? 'in_progress' : 'completed'
      )
    },
    onSuccess: (_, action) => {
      if (action === 'reply') setBody('')
      toast.success(t('Saved successfully'))
      props.refresh()
    },
    onError: (error) => {
      handleServerError(error)
      props.refresh()
    },
  })
  return (
    <div className='space-y-5'>
      <div className='flex items-center gap-2'>
        <span className='text-muted-foreground'>#{row.id}</span>
        <TicketStatusBadge status={row.status} />
      </div>
      <dl className='grid grid-cols-2 gap-3 text-sm'>
        <div>
          <dt className='text-muted-foreground'>{t('Submitted by')}</dt>
          <dd>{row.owner_name || `#${row.owner_id}`}</dd>
        </div>
        <div>
          <dt className='text-muted-foreground'>{t('Assigned to')}</dt>
          <dd>
            {row.assignee_name ||
              (row.assignee_id ? `#${row.assignee_id}` : t('None'))}
          </dd>
        </div>
        <div>
          <dt className='text-muted-foreground'>{t('Created')}</dt>
          <dd>{formatTimestamp(row.created_at)}</dd>
        </div>
        <div>
          <dt className='text-muted-foreground'>{t('Updated')}</dt>
          <dd>{formatTimestamp(row.updated_at)}</dd>
        </div>
      </dl>
      <p className='break-words whitespace-pre-wrap'>{row.description}</p>
      {row.attachment_name && (
        <Button
          variant='outline'
          onClick={() =>
            void downloadTicket(props.all, row).catch(handleServerError)
          }
        >
          {row.attachment_name}
        </Button>
      )}
      <section className='space-y-2 border-t pt-4'>
        <h3 className='font-medium'>{t('Timeline')}</h3>
        {row.entries.length === 0 && (
          <p className='text-muted-foreground text-sm'>{t('No records')}</p>
        )}
        <ol className='space-y-3'>
          {row.entries.map((entry) => (
            <li key={entry.id} className='border-l-2 pl-3'>
              <div className='text-sm'>
                <strong>{entry.actor_name || `#${entry.actor_id}`}</strong> ·{' '}
                {t(entryLabels[entry.action])} ·{' '}
                {formatTimestamp(entry.created_at)}
              </div>
              {entry.body && (
                <p className='mt-1 break-words whitespace-pre-wrap'>
                  {entry.body}
                </p>
              )}
            </li>
          ))}
        </ol>
      </section>
      {canReply && (
        <form
          className='space-y-2'
          onSubmit={(e) => {
            e.preventDefault()
            if (body.trim()) mutation.mutate('reply')
          }}
        >
          <Label htmlFor='ticket-reply'>{t('Reply')}</Label>
          <Textarea
            id='ticket-reply'
            required
            maxLength={4000}
            value={body}
            onChange={(e) => setBody(e.target.value)}
          />
          <Button
            type='submit'
            variant='outline'
            disabled={!body.trim() || mutation.isPending}
          >
            {t('Send reply')}
          </Button>
        </form>
      )}
      <div className='flex justify-end gap-2'>
        {isOwner && row.status === 'pending' && (
          <Button variant='outline' onClick={props.edit}>
            {t('Edit ticket')}
          </Button>
        )}
        {canClaim && (
          <Button
            disabled={mutation.isPending}
            onClick={() => mutation.mutate('claim')}
          >
            {t('Claim ticket')}
          </Button>
        )}
        {handler && row.status === 'in_progress' && (
          <Button
            disabled={mutation.isPending}
            onClick={() => mutation.mutate('complete')}
          >
            {t('Complete ticket')}
          </Button>
        )}
      </div>
    </div>
  )
}
