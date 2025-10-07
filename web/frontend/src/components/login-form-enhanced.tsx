import { useForm } from '@tanstack/react-form'
import { z } from 'zod'
import { EyeIcon, EyeOffIcon, Loader2Icon } from 'lucide-react'
import { Link } from '@tanstack/react-router'
import { useEffect, useState } from 'react'
import { cn } from '@/lib/utils'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Checkbox } from '@/components/ui/checkbox'
import { useLogin } from '@/lib/use-login'

const loginSchema = z.object({
  email: z.string().min(1, 'Email is required').email('Invalid email address'),
  password: z
    .string()
    .min(1, 'Password is required')
    .min(6, 'Password must be at least 6 characters'),
  rememberMe: z.boolean().optional(),
})

type LoginFormData = z.infer<typeof loginSchema>

function FieldError({ error }: { error?: string | Array<string> }) {
  if (!error) return null
  const errorMessage = Array.isArray(error) ? error[0] : error
  return (
    <p className="text-sm text-destructive mt-1 animate-in fade-in-50 slide-in-from-top-1">
      {errorMessage}
    </p>
  )
}

function usePersistedEmail() {
  const [email, setEmail] = useState('')

  useEffect(() => {
    const stored = localStorage.getItem('remembered_email')
    if (stored) setEmail(stored)
  }, [])

  const saveEmail = (email: string, remember: boolean) => {
    if (remember) {
      localStorage.setItem('remembered_email', email)
    } else {
      localStorage.removeItem('remembered_email')
    }
  }

  return { email, saveEmail }
}

export function LoginFormEnhanced({
  className,
  onSuccess,
  redirectTo = '/dashboard',
  ...props
}: React.ComponentProps<'div'> & {
  onSuccess?: () => void
  redirectTo?: string
}) {
  const loginMutation = useLogin()
  const [serverError, setServerError] = useState<string | null>(null)
  const [showPassword, setShowPassword] = useState(false)
  const { email: persistedEmail, saveEmail } = usePersistedEmail()

  const form = useForm({
    defaultValues: {
      email: persistedEmail,
      password: '',
      rememberMe: !!persistedEmail,
    } as LoginFormData,
    onSubmit: async ({ value }) => {
      try {
        setServerError(null)

        saveEmail(value.email, value.rememberMe || false)

        await loginMutation.mutateAsync({
          email: value.email,
          password: value.password,
        })

        onSuccess?.()

        if (typeof window !== 'undefined') {
          window.location.href = redirectTo
        }
      } catch (error) {
        if (error instanceof Error) {
          if (error.message.includes('401')) {
            setServerError('Invalid email or password')
          } else if (error.message.includes('network')) {
            setServerError('Network error. Please check your connection.')
          } else {
            setServerError(error.message)
          }
        } else {
          setServerError('An unexpected error occurred. Please try again.')
        }
      }
    },
  })

  // Email field validator
  const validateEmail = (value: string) => {
    if (!value) return 'Email is required'
    if (!/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(value)) {
      return 'Please enter a valid email address'
    }
    return undefined
  }

  const validatePassword = (value: string) => {
    if (!value) return 'Password is required'
    if (value.length < 6) return 'Password must be at least 6 characters'
    return undefined
  }

  return (
    <div
      className={cn('flex flex-col gap-6 w-full max-w-sm', className)}
      {...props}
    >
      <div className="flex flex-col items-center gap-2 text-center">
        <h1 className="text-2xl font-bold">Welcome back</h1>
        <p className="text-muted-foreground text-sm text-balance">
          Enter your credentials to access your account
        </p>
      </div>

      <form
        onSubmit={(e) => {
          e.preventDefault()
          e.stopPropagation()
          form.handleSubmit()
        }}
        className="grid gap-4"
      >
        <div className="grid gap-2">
          <form.Field
            name="email"
            validators={{
              onChange: ({ value }) => validateEmail(value),
              onChangeAsyncDebounceMs: 500,
            }}
            children={(field) => (
              <>
                <Label htmlFor={field.name}>
                  Email
                  <span className="text-destructive ml-1">*</span>
                </Label>
                <div className="relative">
                  <Input
                    id={field.name}
                    type="email"
                    placeholder="name@example.com"
                    value={field.state.value}
                    onBlur={field.handleBlur}
                    onChange={(e) => field.handleChange(e.target.value)}
                    className={cn(
                      'pr-10',
                      field.state.meta.isTouched &&
                        field.state.meta.errors.length > 0 &&
                        'border-destructive focus-visible:ring-destructive',
                    )}
                    aria-invalid={field.state.meta.errors.length > 0}
                    aria-describedby={
                      field.state.meta.errors.length > 0
                        ? `${field.name}-error`
                        : undefined
                    }
                    autoComplete="email"
                  />
                  {field.state.meta.isTouched &&
                    field.state.value &&
                    !field.state.meta.errors.length && (
                      <span className="absolute right-3 top-1/2 -translate-y-1/2 text-green-500">
                        ✓
                      </span>
                    )}
                </div>
                <FieldError
                  error={
                    field.state.meta.isTouched
                      ? field.state.meta.errors[0]
                      : undefined
                  }
                />
              </>
            )}
          />
        </div>

        <div className="grid gap-2">
          <form.Field
            name="password"
            validators={{
              onChange: ({ value }) => validatePassword(value),
            }}
            children={(field) => (
              <>
                <div className="flex items-center justify-between">
                  <Label htmlFor={field.name}>
                    Password
                    <span className="text-destructive ml-1">*</span>
                  </Label>
                  <Link
                    to="/"
                    className="text-sm text-primary underline-offset-4 hover:underline"
                  >
                    Forgot password?
                  </Link>
                </div>
                <div className="relative">
                  <Input
                    id={field.name}
                    type={showPassword ? 'text' : 'password'}
                    value={field.state.value}
                    onBlur={field.handleBlur}
                    onChange={(e) => field.handleChange(e.target.value)}
                    className={cn(
                      'pr-10',
                      field.state.meta.isTouched &&
                        field.state.meta.errors.length > 0 &&
                        'border-destructive focus-visible:ring-destructive',
                    )}
                    aria-invalid={field.state.meta.errors.length > 0}
                    aria-describedby={
                      field.state.meta.errors.length > 0
                        ? `${field.name}-error`
                        : undefined
                    }
                    autoComplete="current-password"
                  />
                  <button
                    type="button"
                    onClick={() => setShowPassword(!showPassword)}
                    className="absolute right-3 top-1/2 -translate-y-1/2 text-muted-foreground hover:text-foreground transition-colors"
                    aria-label={
                      showPassword ? 'Hide password' : 'Show password'
                    }
                  >
                    {showPassword ? (
                      <EyeOffIcon className="h-4 w-4" />
                    ) : (
                      <EyeIcon className="h-4 w-4" />
                    )}
                  </button>
                </div>
                <FieldError
                  error={
                    field.state.meta.isTouched
                      ? field.state.meta.errors[0]
                      : undefined
                  }
                />
              </>
            )}
          />
        </div>

        <div className="flex items-center space-x-2">
          <form.Field
            name="rememberMe"
            children={(field) => (
              <>
                <Checkbox
                  id={field.name}
                  checked={field.state.value}
                  onCheckedChange={(checked) =>
                    field.handleChange(checked === true)
                  }
                />
                <Label
                  htmlFor={field.name}
                  className="text-sm font-normal cursor-pointer select-none"
                >
                  Remember me for 30 days
                </Label>
              </>
            )}
          />
        </div>

        {serverError && (
          <div className="text-center justify-center bg-destructive/10 border border-destructive/20 rounded-md p-3 animate-in fade-in-50 slide-in-from-top-1">
            <p className="text-sm text-center justify-center text-destructive flex items-center gap-2">
              {serverError}
            </p>
          </div>
        )}

        {/* Submit Button */}
        <form.Subscribe
          selector={(state) => [state.canSubmit, state.isSubmitting]}
          children={([canSubmit, isSubmitting]) => (
            <Button
              type="submit"
              className="w-full"
              disabled={!canSubmit || isSubmitting}
            >
              {isSubmitting ? (
                <>
                  <Loader2Icon className="mr-2 h-4 w-4 animate-spin" />
                  Signing in...
                </>
              ) : (
                'Sign in'
              )}
            </Button>
          )}
        />

        {/* Divider */}
        <div className="relative">
          <div className="absolute inset-0 flex items-center">
            <span className="w-full border-t" />
          </div>
          <div className="relative flex justify-center text-xs uppercase">
            <span className="bg-background px-2 text-muted-foreground">
              Or continue with
            </span>
          </div>
        </div>

        {/* Social Login Buttons */}
        <div className="grid grid-cols-2 gap-3">
          <Button
            type="button"
            variant="outline"
            disabled={form.state.isSubmitting}
            onClick={() => {
              console.log('GitHub login')
              // Implement GitHub OAuth flow
            }}
          >
            <svg
              className="mr-2 h-4 w-4"
              aria-hidden="true"
              focusable="false"
              data-prefix="fab"
              data-icon="github"
              role="img"
              xmlns="http://www.w3.org/2000/svg"
              viewBox="0 0 496 512"
            >
              <path
                fill="currentColor"
                d="M165.9 397.4c0 2-2.3 3.6-5.2 3.6-3.3.3-5.6-1.3-5.6-3.6 0-2 2.3-3.6 5.2-3.6 3-.3 5.6 1.3 5.6 3.6zm-31.1-4.5c-.7 2 1.3 4.3 4.3 4.9 2.6 1 5.6 0 6.2-2s-1.3-4.3-4.3-5.2c-2.6-.7-5.5.3-6.2 2.3zm44.2-1.7c-2.9.7-4.9 2.6-4.6 4.9.3 2 2.9 3.3 5.9 2.6 2.9-.7 4.9-2.6 4.6-4.6-.3-1.9-3-3.2-5.9-2.9zM244.8 8C106.1 8 0 113.3 0 252c0 110.9 69.8 205.8 169.5 239.2 12.8 2.3 17.3-5.6 17.3-12.1 0-6.2-.3-40.4-.3-61.4 0 0-70 15-84.7-29.8 0 0-11.4-29.1-27.8-36.6 0 0-22.9-15.7 1.6-15.4 0 0 24.9 2 38.6 25.8 21.9 38.6 58.6 27.5 72.9 20.9 2.3-16 8.8-27.1 16-33.7-55.9-6.2-112.3-14.3-112.3-110.5 0-27.5 7.6-41.3 23.6-58.9-2.6-6.5-11.1-33.3 2.6-67.9 20.9-6.5 69 27 69 27 20-5.6 41.5-8.5 62.8-8.5s42.8 2.9 62.8 8.5c0 0 48.1-33.6 69-27 13.7 34.7 5.2 61.4 2.6 67.9 16 17.7 25.8 31.5 25.8 58.9 0 96.5-58.9 104.2-114.8 110.5 9.2 7.9 17 22.9 17 46.4 0 33.7-.3 75.4-.3 83.6 0 6.5 4.6 14.4 17.3 12.1C428.2 457.8 496 362.9 496 252 496 113.3 383.5 8 244.8 8zM97.2 352.9c-1.3 1-1 3.3.7 5.2 1.6 1.6 3.9 2.3 5.2 1 1.3-1 1-3.3-.7-5.2-1.6-1.6-3.9-2.3-5.2-1zm-10.8-8.1c-.7 1.3.3 2.9 2.3 3.9 1.6 1 3.6.7 4.3-.7.7-1.3-.3-2.9-2.3-3.9-2-.6-3.6-.3-4.3.7zm32.4 35.6c-1.6 1.3-1 4.3 1.3 6.2 2.3 2.3 5.2 2.6 6.5 1 1.3-1.3.7-4.3-1.3-6.2-2.2-2.3-5.2-2.6-6.5-1zm-11.4-14.7c-1.6 1-1.6 3.6 0 5.9 1.6 2.3 4.3 3.3 5.6 2.3 1.6-1.3 1.6-3.9 0-6.2-1.4-2.3-4-3.3-5.6-2z"
              />
            </svg>
            GitHub
          </Button>

          <Button
            type="button"
            variant="outline"
            disabled={form.state.isSubmitting}
            onClick={() => {
              console.log('Google login')
              // Implement Google OAuth flow
            }}
          >
            <svg
              className="mr-2 h-4 w-4"
              aria-hidden="true"
              focusable="false"
              data-prefix="fab"
              data-icon="google"
              role="img"
              xmlns="http://www.w3.org/2000/svg"
              viewBox="0 0 488 512"
            >
              <path
                fill="currentColor"
                d="M488 261.8C488 403.3 391.1 504 248 504 110.8 504 0 393.2 0 256S110.8 8 248 8c66.8 0 123 24.5 166.3 64.9l-67.5 64.9C258.5 52.6 94.3 116.6 94.3 256c0 86.5 69.1 156.6 153.7 156.6 98.2 0 135-70.4 140.8-106.9H248v-85.3h236.1c2.3 12.7 3.9 24.9 3.9 41.4z"
              />
            </svg>
            Google
          </Button>
        </div>
      </form>

      {/* Sign up link */}
      <div className="text-center text-sm">
        <span className="text-muted-foreground">Don't have an account? </span>
        <Link
          to="/register"
          className="font-medium text-primary underline-offset-4 hover:underline"
        >
          Create account
        </Link>
      </div>
    </div>
  )
}
