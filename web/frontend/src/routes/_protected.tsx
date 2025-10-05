// eslint-disable-next-line
import { createFileRoute, Link, Outlet } from '@tanstack/react-router'
import { requireAuthAndOrg } from '@/lib/route-guards'
import { Navbar } from '@/components/navbar'

export const Route = createFileRoute('/_protected')({
  beforeLoad: async ({ context, location }) => {
    const authState = await requireAuthAndOrg({ location })

    return {
      auth: authState,
    }
  },

  component: ProtectedLayout,
})

function ProtectedLayout() {
  return (
    <div className="min-h-screen bg-background">
      <Navbar />

      <main className="container mx-auto px-4 py-6">
        <Outlet />
      </main>

      <footer className="mt-auto border-t">
        <div className="container mx-auto px-4 py-4">
          <div className="flex flex-col sm:flex-row justify-between items-center gap-4 text-sm text-muted-foreground">
            <p>© 2024 ZoriHQ. All rights reserved.</p>
            <div className="flex gap-4">
              <Link
                to="/about"
                className="hover:text-foreground transition-colors"
              >
                About
              </Link>
              <Link to="/" className="hover:text-foreground transition-colors">
                Privacy
              </Link>
              <Link to="/" className="hover:text-foreground transition-colors">
                Terms
              </Link>
            </div>
          </div>
        </div>
      </footer>
    </div>
  )
}
