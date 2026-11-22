import { useMutation } from '@tanstack/react-query'
import { useForm } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { Dialog } from '@/components/dialog'
import { Button } from '@/components/ui/button'
import {
  Form,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
  FormControl,
} from '@/components/ui/form'
import { Input } from '@/components/ui/input'
import { Textarea } from '@/components/ui/textarea'
import { handleServerError } from '@/lib/handle-server-error'

import { saveTicket, type Ticket } from './api'

type TicketFields = {
  title: string
  description: string
  attachment?: FileList
}

export function TicketEditor({
  ticket,
  limit,
  close,
  saved,
}: {
  limit: number
  ticket?: Ticket
  close: () => void
  saved: () => void
}) {
  const { t } = useTranslation()
  const form = useForm<TicketFields>({
    defaultValues: {
      title: ticket?.title ?? '',
      description: ticket?.description ?? '',
    },
  })
  const mutation = useMutation({
    mutationFn: async (values: TicketFields) => {
      const body = new FormData()
      body.set('title', values.title.trim())
      body.set('description', values.description.trim())
      if (ticket) body.set('version', String(ticket.version))
      if (values.attachment?.[0]) body.set('attachment', values.attachment[0])
      await saveTicket(body, ticket?.id)
    },
    onSuccess: () => {
      toast.success(t('Saved successfully'))
      saved()
    },
    onError: (error) => handleServerError(error),
  })
  return (
    <Dialog
      open
      title={ticket ? t('Edit ticket') : t('Create ticket')}
      onOpenChange={(open) => {
        if (!open && !mutation.isPending) close()
      }}
    >
      <Form {...form}>
        <form
          className='space-y-4'
          onSubmit={form.handleSubmit((v) => mutation.mutate(v))}
        >
          <FormField
            control={form.control}
            name='title'
            rules={{ validate: (v) => v.trim().length > 0 || t('Required') }}
            render={({ field }) => (
              <FormItem>
                <FormLabel>{t('Title')}</FormLabel>
                <FormControl>
                  <Input maxLength={160} {...field} />
                </FormControl>
                <FormMessage />
              </FormItem>
            )}
          />
          <FormField
            control={form.control}
            name='description'
            rules={{ validate: (v) => v.trim().length > 0 || t('Required') }}
            render={({ field }) => (
              <FormItem>
                <FormLabel>{t('Description')}</FormLabel>
                <FormControl>
                  <Textarea rows={5} maxLength={4000} {...field} />
                </FormControl>
                <FormMessage />
              </FormItem>
            )}
          />
          <FormItem>
            <label htmlFor='ticket-attachment'>
              {t('Attachment (up to {{limit}} KiB)', { limit })}
            </label>
            <Input
              id='ticket-attachment'
              type='file'
              {...form.register('attachment', {
                validate: (files) =>
                  (files?.[0]?.size ?? 0) <= limit * 1024 ||
                  t('Attachment is too large'),
              })}
            />
            {form.formState.errors.attachment?.message && (
              <p role='alert'>{form.formState.errors.attachment.message}</p>
            )}
          </FormItem>
          <div className='flex justify-end gap-2'>
            <Button
              type='button'
              variant='outline'
              onClick={close}
              disabled={mutation.isPending}
            >
              {t('Cancel')}
            </Button>
            <Button type='submit' disabled={mutation.isPending}>
              {t('Save')}
            </Button>
          </div>
        </form>
      </Form>
    </Dialog>
  )
}
