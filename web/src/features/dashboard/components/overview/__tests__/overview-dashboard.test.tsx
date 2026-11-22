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
import {
  createMemoryHistory,
  createRootRoute,
  createRouter,
  RouterProvider,
} from '@tanstack/react-router'
import { cleanup, render, screen } from '@testing-library/react'
import { afterEach, expect, it, vi } from 'vitest'

import { api } from '@/lib/api'
import { useAuthStore } from '@/stores/auth-store'

import { OverviewDashboard } from '../overview-dashboard'

afterEach(() => {
  cleanup()
  useAuthStore.setState(useAuthStore.getInitialState(), true)
  vi.restoreAllMocks()
})

vi.mock('@/hooks/use-status', () => ({ useStatus: () => ({ status: null }) }))
vi.mock('@/business-overview', () => ({ businessOverviews: [] }))

it('shows the overview without a welcome card or a gateway request', async () => {
  useAuthStore.getState().auth.setUser({
    id: 1,
    username: 'dashboard-user',
    display_name: 'Dashboard User',
    role: 1,
  })
  const get = vi.spyOn(api, 'get')
  const router = createRouter({
    routeTree: createRootRoute({ component: OverviewDashboard }),
    history: createMemoryHistory({ initialEntries: ['/'] }),
  })
  await router.load()
  render(<RouterProvider router={router} />)

  expect(await screen.findByRole('heading', { name: 'Overview' })).toBeVisible()
  expect(screen.getAllByRole('heading')).toHaveLength(1)
  expect(screen.queryByText(/Dashboard User/)).not.toBeInTheDocument()
  expect(get).not.toHaveBeenCalled()
})
