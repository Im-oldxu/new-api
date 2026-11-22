import { useMutation, useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { Dialog } from '@/components/dialog'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'

import { updateTicketSettings } from './api'

export function TicketSettingsDialog(props: {
  limit: number
  close: () => void
}) {
  const { t } = useTranslation()
  const client = useQueryClient()
  const [value, setValue] = useState(String(props.limit))
  const save = useMutation({
    mutationFn: updateTicketSettings,
    onSuccess: (result) => {
      client.setQueryData(['ticket-settings'], result)
      toast.success(t('Saved successfully'))
      props.close()
    },
  })
  return (
    <Dialog
      open
      title={t('Ticket settings')}
      onOpenChange={(open) => {
        if (!open && !save.isPending) props.close()
      }}
    >
      <form
        className='space-y-4'
        onSubmit={(e) => {
          e.preventDefault()
          save.mutate(Number(value))
        }}
      >
        <Label htmlFor='ticket-limit'>{t('Attachment limit (KiB)')}</Label>
        <Input
          id='ticket-limit'
          type='number'
          min={1}
          max={1024}
          step={1}
          required
          value={value}
          onChange={(e) => setValue(e.target.value)}
        />
        {save.isError && <p role='alert'>{t('Update failed')}</p>}
        <Button type='submit' disabled={save.isPending}>
          {t('Save')}
        </Button>
      </form>
    </Dialog>
  )
}
