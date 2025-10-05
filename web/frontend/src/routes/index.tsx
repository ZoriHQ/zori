import { createFileRoute, redirect } from '@tanstack/react-router'

export const Route = createFileRoute('/')({
  beforeLoad: async () => {
    // Redirect root to dashboard
    throw redirect({
      to: '/dashboard',
      replace: true,
    })
  },
})
