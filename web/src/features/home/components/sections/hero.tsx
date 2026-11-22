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
import { Link } from '@tanstack/react-router'
import { BookOpen } from 'lucide-react'
import { useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'
import { useStatus } from '@/hooks/use-status'
import { DEFAULT_SYSTEM_NAME } from '@/lib/constants'

import { HeroButtons } from '../hero-buttons'

interface HeroProps {
  isAuthenticated: boolean
}

export function Hero({ isAuthenticated }: HeroProps) {
  const { t } = useTranslation()
  const { status } = useStatus()
  const docsUrl = typeof status?.docs_link === 'string' && status.docs_link

  return (
    <section className='px-6 py-24 md:py-32'>
      <div className='mx-auto flex max-w-6xl flex-col items-start gap-8'>
        <h1 className='text-4xl font-bold tracking-tight md:text-5xl'>
          {status?.system_name || DEFAULT_SYSTEM_NAME}
        </h1>
        <div className='flex flex-wrap items-center gap-3'>
          <HeroButtons isAuthenticated={isAuthenticated} />
          {docsUrl && (
            <Button
              size='lg'
              variant='outline'
              render={
                docsUrl.startsWith('http') ? (
                  <a href={docsUrl} target='_blank' rel='noopener noreferrer' />
                ) : (
                  <Link to={docsUrl} />
                )
              }
            >
              <BookOpen aria-hidden /> {t('Docs')}
            </Button>
          )}
        </div>
      </div>
    </section>
  )
}
