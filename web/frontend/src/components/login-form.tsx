'use client'

import { useForm } from '@tanstack/react-form'
import { z } from 'zod'
import { cn } from '@/lib/utils'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { useLogin } from '@/lib/use-login'
import { useState } from 'react'

// Define validation schema
const loginSchema = z.object({
  email: z.string().min(1, 'Email is required').email('Invalid email address'),
  password: z
    .string()
    .min(1, 'Password is required')
    .min(6, 'Password must be at least 6 characters'),
})

type LoginFormData = z.infer<typeof loginSchema>

export function LoginForm({
  className,
  ...props
}: React.ComponentProps<'div'>) {
  const loginMutation = useLogin()
  const [serverError, setServerError] = useState<string | null>(null)

  const form = useForm({
    defaultValues: {
      email: '',
      password: '',
    } as LoginFormData,
    onSubmit: async ({ value }) => {
      try {
        setServerError(null)
        await loginMutation.mutateAsync(value)
      } catch (error) {
        setServerError(
          error instanceof Error
            ? error.message
            : 'An error occurred during login',
        )
      }
    },
    validators: {
      onChange: ({ value }) => {
        try {
          loginSchema.parse(value)
          return undefined
        } catch (error) {
          if (error instanceof z.ZodError) {
            return error.format()
          }
          return undefined
        }
      },
    },
  })

  return (
    <div className={cn('flex flex-col gap-6', className)} {...props}>
      <div className="flex flex-col items-center gap-2 text-center">
        <h1 className="text-2xl font-bold">Login to your account</h1>
        <p className="text-muted-foreground text-sm text-balance">
          Enter your email below to login to your account
        </p>
      </div>

      <form
        onSubmit={(e) => {
          e.preventDefault()
          e.stopPropagation()
          form.handleSubmit()
        }}
        className="grid gap-6"
      >
        <div className="grid gap-3">
          <form.Field
            name="email"
            validators={{
              onChange: ({ value }) => {
                try {
                  loginSchema.shape.email.parse(value)
                  return undefined
                } catch (error) {
                  if (error instanceof z.ZodError) {
                    return error.errors[0]?.message
                  }
                  return undefined
                }
              },
              onChangeAsyncDebounceMs: 500,
              onChangeAsync: async ({ value }) => {
                // You can add async validation here if needed
                // For example, checking if email exists in the database
                return undefined
              },
            }}
            children={(field) => (
              <>
                <Label htmlFor={field.name}>Email</Label>
                <Input
                  id={field.name}
                  type="email"
                  placeholder="m@example.com"
                  value={field.state.value}
                  onBlur={field.handleBlur}
                  onChange={(e) => field.handleChange(e.target.value)}
                  className={cn(
                    field.state.meta.errors.length > 0 && 'border-destructive',
                  )}
                  aria-invalid={field.state.meta.errors.length > 0}
                  aria-describedby={
                    field.state.meta.errors.length > 0
                      ? `${field.name}-error`
                      : undefined
                  }
                />
                {field.state.meta.errors.length > 0 && (
                  <p
                    id={`${field.name}-error`}
                    className="text-sm text-destructive mt-1"
                  >
                    {field.state.meta.errors.join(', ')}
                  </p>
                )}
              </>
            )}
          />
        </div>

        <div className="grid gap-3">
          <form.Field
            name="password"
            validators={{
              onChange: ({ value }) => {
                try {
                  loginSchema.shape.password.parse(value)
                  return undefined
                } catch (error) {
                  if (error instanceof z.ZodError) {
                    return error.errors[0]?.message
                  }
                  return undefined
                }
              },
            }}
            children={(field) => (
              <>
                <div className="flex items-center">
                  <Label htmlFor={field.name}>Password</Label>
                  <a
                    href="#"
                    className="ml-auto text-sm underline-offset-4 hover:underline"
                  >
                    Forgot your password?
                  </a>
                </div>
                <Input
                  id={field.name}
                  type="password"
                  value={field.state.value}
                  onBlur={field.handleBlur}
                  onChange={(e) => field.handleChange(e.target.value)}
                  className={cn(
                    field.state.meta.errors.length > 0 && 'border-destructive',
                  )}
                  aria-invalid={field.state.meta.errors.length > 0}
                  aria-describedby={
                    field.state.meta.errors.length > 0
                      ? `${field.name}-error`
                      : undefined
                  }
                />
                {field.state.meta.errors.length > 0 && (
                  <p
                    id={`${field.name}-error`}
                    className="text-sm text-destructive mt-1"
                  >
                    {field.state.meta.errors.join(', ')}
                  </p>
                )}
              </>
            )}
          />
        </div>

        {serverError && (
          <div className="bg-destructive/10 border border-destructive/20 rounded-md p-3">
            <p className="text-sm text-destructive">{serverError}</p>
          </div>
        )}

        <form.Subscribe
          selector={(state) => [state.canSubmit, state.isSubmitting]}
          children={([canSubmit, isSubmitting]) => (
            <Button
              type="submit"
              className="w-full"
              disabled={!canSubmit || isSubmitting}
            >
              {isSubmitting ? 'Logging in...' : 'Login'}
            </Button>
          )}
        />

        <div className="after:border-border relative text-center text-sm after:absolute after:inset-0 after:top-1/2 after:z-0 after:flex after:items-center after:border-t">
          <span className="bg-background text-muted-foreground relative z-10 px-2">
            Or continue with
          </span>
        </div>

        <Button
          type="button"
          variant="outline"
          className="w-full"
          disabled={form.state.isSubmitting}
          onClick={() => {
            // Handle GitHub login
            console.log('GitHub login clicked')
          }}
        >
          <svg
            xmlns="http://www.w3.org/2000/svg"
            viewBox="0 0 24 24"
            className="mr-2 h-4 w-4"
          >
            <path
              d="M12 .297c-6.63 0-12 5.373-12 12 0 5.303 3.438 9.8 8.205 11.385.6.113.82-.258.82-.577 0-.285-.01-1.04-.015-2.04-3.338.724-4.042-1.61-4.042-1.61C4.422 18.07 3.633 17.7 3.633 17.7c-1.087-.744.084-.729.084-.729 1.205.084 1.838 1.236 1.838 1.236 1.07 1.835 2.809 1.305 3.495.998.108-.776.417-1.305.76-1.605-2.665-.3-5.466-1.332-5.466-5.93 0-1.31.465-2.38 1.235-3.22-.135-.303-.54-1.523.105-3.176 0 0 1.005-.322 3.3 1.23.96-.267 1.98-.399 3-.405 1.02.006 2.04.138 3 .405 2.28-1.552 3.285-1.23 3.285-1.23.645 1.653.24 2.873.12 3.176.765.84 1.23 1.91 1.23 3.22 0 4.61-2.805 5.625-5.475 5.92.42.36.81 1.096.81 2.22 0 1.606-.015 2.896-.015 3.286 0 .315.21.69.825.57C20.565 22.092 24 17.592 24 12.297c0-6.627-5.373-12-12-12"
              fill="currentColor"
            />
          </svg>
          Login with GitHub
        </Button>
      </form>

      <div className="text-center text-sm">
        Don&apos;t have an account?{' '}
        <a href="/register" className="underline underline-offset-4">
          Sign up
        </a>
      </div>
    </div>
  )
}
