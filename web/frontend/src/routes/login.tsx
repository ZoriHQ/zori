import { createFileRoute } from '@tanstack/react-router'
import { z } from 'zod'
import LoginPage from '@/app/login/page'
import { getPostLoginRedirect, requireGuest } from '@/lib/route-guards'

const loginSearchSchema = z.object({
  redirect: z.string().optional(),
})

export const Route = createFileRoute('/login')({
  validateSearch: loginSearchSchema,

  beforeLoad: async ({ context, location }) => {
    const authState = await requireGuest({ location })

    return {
      auth: authState,
    }
  },

  component: LoginPageWrapper,
})

function LoginPageWrapper() {
  const search = Route.useSearch()
  const navigate = Route.useNavigate()

  const handleLoginSuccess = () => {
    const redirectTo = getPostLoginRedirect(search)

    navigate({ to: redirectTo })
  }

  return <LoginPage onSuccess={handleLoginSuccess} />
}
