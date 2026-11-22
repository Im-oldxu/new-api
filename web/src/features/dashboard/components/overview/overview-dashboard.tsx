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
import { Suspense } from 'react'
import { useTranslation } from 'react-i18next'

import { businessOverviews } from '@/business-overview'
import { SectionPageLayout } from '@/components/layout'
import { useStatus } from '@/hooks/use-status'

import { AnnouncementsPanel } from './announcements-panel'
import { FAQPanel } from './faq-panel'
import { UptimePanel } from './uptime-panel'

export function OverviewDashboard() {
  const { t } = useTranslation()
  const { status } = useStatus()

  return (
    <SectionPageLayout>
      <SectionPageLayout.Title>{t('Overview')}</SectionPageLayout.Title>
      <SectionPageLayout.Content>
        <div className='grid min-w-0 gap-4'>
          {businessOverviews.map(({ component: Overview, id }) => (
            <Suspense
              key={id}
              fallback={<p role='status'>{t('Loading...')}</p>}
            >
              <Overview />
            </Suspense>
          ))}
          <div className='grid min-w-0 gap-4 empty:hidden lg:grid-cols-2'>
            {status?.announcements_enabled === true &&
              Array.isArray(status.announcements) &&
              status.announcements.length > 0 && <AnnouncementsPanel />}
            {status?.faq_enabled === true &&
              Array.isArray(status.faq) &&
              status.faq.length > 0 && <FAQPanel />}
            {status?.uptime_kuma_enabled === true &&
              status.uptime_configured === true && <UptimePanel />}
          </div>
        </div>
      </SectionPageLayout.Content>
    </SectionPageLayout>
  )
}
