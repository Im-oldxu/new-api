import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { expect, it, vi } from 'vitest'

import { TicketEditor } from '../editor'

it('keeps an oversized attachment in the editor for correction', async () => {
  const user = userEvent.setup()
  const saved = vi.fn()
  render(
    <QueryClientProvider client={new QueryClient()}>
      <TicketEditor limit={1024} close={vi.fn()} saved={saved} />
    </QueryClientProvider>
  )
  await user.type(screen.getByLabelText('Title'), 'Keyboard')
  await user.type(screen.getByLabelText('Description'), 'Attach the receipt')
  await user.upload(
    screen.getByLabelText('Attachment (up to 1024 KiB)'),
    new File([new Uint8Array(1024 * 1024 + 1)], 'receipt.bin')
  )
  await user.click(screen.getByRole('button', { name: 'Save' }))
  expect(await screen.findByRole('alert')).toHaveTextContent('Attachment is too large')
  expect(screen.getByLabelText('Title')).toHaveValue('Keyboard')
  expect(saved).not.toHaveBeenCalled()
})
