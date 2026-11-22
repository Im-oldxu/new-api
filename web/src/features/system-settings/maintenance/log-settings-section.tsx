/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/
import { zodResolver } from '@hookform/resolvers/zod'
import { useEffect, useId, useMemo } from 'react'
import { useForm } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import * as z from 'zod'

import {
  Form,
  FormControl,
  FormDescription,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from '@/components/ui/form'
import { Input } from '@/components/ui/input'
import { Separator } from '@/components/ui/separator'

import {
  SettingsForm,
  SettingsFormGrid,
} from '../components/settings-form-layout'
import { SettingsPageFormActions } from '../components/settings-page-context'
import { SettingsSection } from '../components/settings-section'
import { useUpdateOption } from '../hooks/use-update-option'
import { safeNumberFieldProps } from '../utils/numeric-field'
import { ServerLogFiles } from './server-log-files'

const logSettingsSchema = z.object({
  system_task_setting: z.object({
    retention_days: z.number().int().min(1),
  }),
})

type LogSettingsFormValues = z.infer<typeof logSettingsSchema>

type LogSettingsSectionProps = {
  defaultValues: {
    'system_task_setting.retention_days': number
  }
}

export function LogSettingsSection(props: LogSettingsSectionProps) {
  const { t } = useTranslation()
  const updateOption = useUpdateOption()
  const taskHistoryTitleId = useId()
  const formDefaults = useMemo<LogSettingsFormValues>(
    () => ({
      system_task_setting: {
        retention_days:
          props.defaultValues['system_task_setting.retention_days'],
      },
    }),
    [props.defaultValues]
  )
  const form = useForm<LogSettingsFormValues>({
    resolver: zodResolver(logSettingsSchema),
    defaultValues: formDefaults,
  })

  useEffect(() => {
    form.reset(formDefaults)
  }, [formDefaults, form])

  const onSubmit = async (values: LogSettingsFormValues) => {
    const key = 'system_task_setting.retention_days'
    const value = values.system_task_setting.retention_days
    if (value !== props.defaultValues[key]) {
      await updateOption.mutateAsync({ key, value })
    }
  }

  return (
    <SettingsSection title={t('Log Maintenance')} className='gap-6'>
      <Form {...form}>
        <SettingsForm onSubmit={form.handleSubmit(onSubmit)}>
          <SettingsPageFormActions
            onSave={form.handleSubmit(onSubmit)}
            isSaving={updateOption.isPending}
            saveLabel='Save log settings'
          />
          <div
            role='group'
            aria-labelledby={taskHistoryTitleId}
            className='flex min-w-0 flex-col gap-4'
          >
            <h4 id={taskHistoryTitleId} className='font-medium'>
              {t('System task records')}
            </h4>
            <SettingsFormGrid>
              <FormField
                control={form.control}
                name='system_task_setting.retention_days'
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>{t('Keep finished tasks for (days)')}</FormLabel>
                    <FormControl>
                      <Input
                        type='number'
                        min={1}
                        step={1}
                        placeholder='30'
                        {...safeNumberFieldProps(field)}
                      />
                    </FormControl>
                    <FormDescription>
                      {t(
                        'Finished system tasks older than this are deleted automatically every day.'
                      )}
                    </FormDescription>
                    <FormMessage />
                  </FormItem>
                )}
              />
            </SettingsFormGrid>
          </div>
        </SettingsForm>
      </Form>

      <Separator />

      <ServerLogFiles />
    </SettingsSection>
  )
}
