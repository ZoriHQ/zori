import { Link, createFileRoute } from '@tanstack/react-router'
import { optionalAuth } from '@/lib/route-guards'
import { Button } from '@/components/ui/button'

export const Route = createFileRoute('/about')({
  beforeLoad: async ({ context }) => {
    const authState = await optionalAuth()

    return {
      auth: authState,
    }
  },

  component: AboutPage,
})

function AboutPage() {
  const { auth } = Route.useRouteContext()

  return (
    <div className="container mx-auto px-4 py-8 max-w-4xl">
      <div className="mb-8">
        <h1 className="text-4xl font-bold mb-4">About Us</h1>
        <p className="text-lg text-muted-foreground">
          Welcome to our platform. This is a public page that can be accessed by
          anyone.
        </p>
      </div>

      <div className="prose prose-gray dark:prose-invert max-w-none">
        <h2 className="text-2xl font-semibold mb-4">Our Mission</h2>
        <p className="mb-6">
          We're building the next generation of document processing and
          management tools to help organizations streamline their workflows and
          increase productivity.
        </p>

        <h2 className="text-2xl font-semibold mb-4">Features</h2>
        <ul className="list-disc list-inside space-y-2 mb-6">
          <li>Advanced document processing with AI</li>
          <li>Secure cloud storage</li>
          <li>Real-time collaboration</li>
          <li>Enterprise-grade security</li>
          <li>API integration</li>
        </ul>

        <h2 className="text-2xl font-semibold mb-4">Your Account Status</h2>
        <div className="bg-muted rounded-lg p-6 mb-6">
          {auth.isAuthenticated ? (
            <div>
              <p className="mb-2">
                <span className="font-semibold">Logged in as:</span>{' '}
                {auth.account?.email}
              </p>
              <p className="mb-4">
                <span className="font-semibold">Organization:</span>{' '}
                {auth.organization?.name || 'No organization'}
              </p>
              <div className="flex gap-4">
                <Link to="/">
                  <Button>Go to Dashboard</Button>
                </Link>
                <Link to="/">
                  <Button variant="outline">Account Settings</Button>
                </Link>
              </div>
            </div>
          ) : (
            <div>
              <p className="mb-4">
                You're not logged in. Sign up or log in to access all features.
              </p>
              <div className="flex gap-4">
                <Link to="/register">
                  <Button>Sign Up</Button>
                </Link>
                <Link to="/login">
                  <Button variant="outline">Log In</Button>
                </Link>
              </div>
            </div>
          )}
        </div>

        <h2 className="text-2xl font-semibold mb-4">Contact</h2>
        <p className="mb-6">
          Have questions? Reach out to us at{' '}
          <a
            href="mailto:support@example.com"
            className="text-primary hover:underline"
          >
            support@example.com
          </a>
        </p>
      </div>

      <div className="mt-12 pt-8 border-t">
        <div className="flex justify-between items-center">
          <p className="text-sm text-muted-foreground">
            © 2024 ZoriHQ. All rights reserved.
          </p>
          <div className="flex gap-4">
            <Link
              to="/"
              className="text-sm text-muted-foreground hover:text-foreground"
            >
              Privacy Policy
            </Link>
            <Link
              to="/"
              className="text-sm text-muted-foreground hover:text-foreground"
            >
              Terms of Service
            </Link>
          </div>
        </div>
      </div>
    </div>
  )
}
