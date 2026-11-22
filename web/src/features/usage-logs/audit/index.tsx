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
import { useQueryClient } from '@tanstack/react-query'
import { useCallback, useState } from 'react'
import { useTranslation } from 'react-i18next'

import { SectionPageLayout } from '@/components/layout'
import { Tabs, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { ROLE } from '@/lib/roles'
import { useAuthStore } from '@/stores/auth-store'

import { AuditLogViewer } from './components/audit-log-viewer'

export function AuditLogs() {
  const { t } = useTranslation()
  const user = useAuthStore((state) => state.auth.user)
  const queryClient = useQueryClient()
  const [scope, setScope] = useState<'all' | 'self'>('all')
  const [accessRevoked, setAccessRevoked] = useState(false)
  const userId = user?.id
  const handleAccessDenied = useCallback(async () => {
    setAccessRevoked(true)
    await queryClient.cancelQueries({ queryKey: ['audit', userId] })
    queryClient.removeQueries({ queryKey: ['audit', userId] })
  }, [queryClient, userId])
  if (!user || user.role < ROLE.ADMIN || accessRevoked) {
    return <p role='alert'>{t('Permission denied')}</p>
  }
  return (
    <SectionPageLayout fixedContent>
      <SectionPageLayout.Title>{t('Audit Logs')}</SectionPageLayout.Title>
      <SectionPageLayout.Actions>
        {
          <Tabs
            value={scope}
            onValueChange={(value) =>
              setScope(value === 'all' ? 'all' : 'self')
            }
          >
            <TabsList aria-label={t('View scope')}>
              <TabsTrigger value='all'>{t('All')}</TabsTrigger>
              <TabsTrigger value='self'>{t('Only Mine')}</TabsTrigger>
            </TabsList>
          </Tabs>
        }
      </SectionPageLayout.Actions>
      <SectionPageLayout.Content>
        <div className='flex h-full min-h-0 flex-col gap-3'>
          <div className='min-h-0 flex-1'>
            <AuditLogViewer
              key={`${userId}:${user.role}:${scope}`}
              scope={scope}
              onAccessDenied={handleAccessDenied}
            />
          </div>
        </div>
      </SectionPageLayout.Content>
    </SectionPageLayout>
  )
}
