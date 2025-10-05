import { createFileRoute } from '@tanstack/react-router'
import { z } from 'zod'
import RegisterPage from '@/app/register/page'
import { getPostLoginRedirect, requireGuest } from '@/lib/route-guards'

const registerSearchSchema = z.object({
  redirect: z.string().optional(),
})

export const Route = createFileRoute('/register')({
  validateSearch: registerSearchSchema,

  beforeLoad: async ({ context, location }) => {
    const authState = await requireGuest({ location })

    return {
      auth: authState,
    }
  },

  component: RegisterPageWrapper,
})

function RegisterPageWrapper() {
  const search = Route.useSearch()
  const navigate = Route.useNavigate()

  const handleRegisterSuccess = () => {
    const redirectTo = getPostLoginRedirect(search)

    navigate({ to: redirectTo })
  }

  return <RegisterPage onSuccess={handleRegisterSuccess} />
}
