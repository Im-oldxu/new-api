import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import {
  createMemoryHistory,
  createRootRoute,
  createRouter,
  RouterProvider,
} from '@tanstack/react-router'
import { cleanup, render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, expect, it, vi } from 'vitest'

import { useAuthStore } from '@/stores/auth-store'

import * as tickets from '../api'
import TicketOverview from '../overview'

afterEach(() => {
  cleanup()
  useAuthStore.setState(useAuthStore.getInitialState(), true)
  vi.restoreAllMocks()
})

it.each([1, 100])(
  'uses the allowed statistics scope for role %s',
  async (role) => {
    useAuthStore.getState().auth.setUser({ id: 1, username: 'viewer', role })
    const fetchOverview = vi
      .spyOn(tickets, 'getTicketOverview')
      .mockResolvedValue({
        total: 0,
        counts: { pending: 0, in_progress: 0, completed: 0 },
        start: 0,
        trend: [],
        recent: [],
      })
    const router = createRouter({
      routeTree: createRootRoute({ component: TicketOverview }),
      history: createMemoryHistory({ initialEntries: ['/'] }),
    })
    await router.load()
    render(
      <QueryClientProvider client={new QueryClient()}>
        <RouterProvider router={router} />
      </QueryClientProvider>
    )
    expect(
      await screen.findByRole('heading', { name: 'Status distribution' })
    ).toBeVisible()
    expect(fetchOverview).toHaveBeenCalledWith(role === 100, 7)
    if (role === 1) {
      expect(
        screen.queryByRole('combobox', { name: 'Scope' })
      ).not.toBeInTheDocument()
      return
    }
    await userEvent.selectOptions(
      screen.getByRole('combobox', { name: 'Scope' }),
      'self'
    )
    expect(await screen.findByRole('combobox', { name: 'Scope' })).toHaveValue(
      'self'
    )
    expect(fetchOverview).toHaveBeenLastCalledWith(false, 7)
  }
)
