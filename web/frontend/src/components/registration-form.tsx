'use client'

import { useState } from 'react'
import { useForm } from '@tanstack/react-form'
import { EyeIcon, EyeOffIcon, Loader2Icon } from 'lucide-react'
import { z } from 'zod'
import { Link } from '@tanstack/react-router'
import { cn } from '@/lib/utils'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { useRegister } from '@/lib/use-register'

// Define validation schema
const registrationSchema = z
  .object({
    firstName: z
      .string()
      .min(1, 'First name is required')
      .min(2, 'First name must be at least 2 characters')
      .max(50, 'First name must be less than 50 characters')
      .regex(
        /^[a-zA-Z\s-']+$/,
        'First name can only contain letters, spaces, hyphens, and apostrophes',
      ),
    lastName: z
      .string()
      .min(1, 'Last name is required')
      .min(2, 'Last name must be at least 2 characters')
      .max(50, 'Last name must be less than 50 characters')
      .regex(
        /^[a-zA-Z\s-']+$/,
        'Last name can only contain letters, spaces, hyphens, and apostrophes',
      ),
    organizationName: z
      .string()
      .min(1, 'Organization name is required')
      .min(2, 'Organization name must be at least 2 characters')
      .max(100, 'Organization name must be less than 100 characters'),
    email: z
      .string()
      .min(1, 'Email is required')
      .email('Invalid email address'),
    password: z
      .string()
      .min(1, 'Password is required')
      .min(8, 'Password must be at least 8 characters')
      .regex(/[A-Z]/, 'Password must contain at least one uppercase letter')
      .regex(/[a-z]/, 'Password must contain at least one lowercase letter')
      .regex(/[0-9]/, 'Password must contain at least one number')
      .regex(
        /[^A-Za-z0-9]/,
        'Password must contain at least one special character',
      ),
    confirmPassword: z.string().min(1, 'Please confirm your password'),
  })
  .refine((data) => data.password === data.confirmPassword, {
    message: "Passwords don't match",
    path: ['confirmPassword'],
  })

type RegistrationFormData = z.infer<typeof registrationSchema>

// Field error component
function FieldError({ error }: { error?: string | Array<string> }) {
  if (!error) return null
  const errorMessage = Array.isArray(error) ? error[0] : error
  return (
    <p className="text-sm text-destructive mt-1 animate-in fade-in-50 slide-in-from-top-1">
      {errorMessage}
    </p>
  )
}

export function RegistrationForm({
  className,
  onSuccess,
  redirectTo = '/dashboard',
  ...props
}: React.ComponentProps<'div'> & {
  onSuccess?: () => void
  redirectTo?: string
}) {
  const registerMutation = useRegister()
  const [serverError, setServerError] = useState<string | null>(null)
  const [showPassword, setShowPassword] = useState(false)
  const [showConfirmPassword, setShowConfirmPassword] = useState(false)

  const form = useForm({
    defaultValues: {
      firstName: '',
      lastName: '',
      organizationName: '',
      email: '',
      password: '',
      confirmPassword: '',
    } as RegistrationFormData,
    onSubmit: async ({ value }) => {
      try {
        setServerError(null)

        // Prepare data without confirmPassword
        const { confirmPassword, ...registrationData } = value

        // Make API call to register using mutation
        await registerMutation.mutateAsync({
          email: registrationData.email,
          password: registrationData.password,
          first_name: registrationData.firstName,
          last_name: registrationData.lastName,
          organization_name: registrationData.organizationName,
        })

        // Call success callback if provided
        onSuccess?.()
      } catch (error) {
        // Handle different error types
        if (error instanceof Error) {
          if (error.message.includes('already exists')) {
            setServerError('An account with this email already exists')
          } else if (error.message.includes('organization')) {
            setServerError('Invalid organization name')
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

  // Field validators
  const validateFirstName = (value: string) => {
    try {
      registrationSchema.shape.firstName.parse(value)
      return undefined
    } catch (error) {
      if (error instanceof z.ZodError) {
        return error.issues[0]?.message
      }
      return undefined
    }
  }

  const validateLastName = (value: string) => {
    try {
      registrationSchema.shape.lastName.parse(value)
      return undefined
    } catch (error) {
      if (error instanceof z.ZodError) {
        return error.issues[0]?.message
      }
      return undefined
    }
  }

  const validateOrganizationName = (value: string) => {
    try {
      registrationSchema.shape.organizationName.parse(value)
      return undefined
    } catch (error) {
      if (error instanceof z.ZodError) {
        return error.issues[0]?.message
      }
      return undefined
    }
  }

  const validateEmail = (value: string) => {
    try {
      registrationSchema.shape.email.parse(value)
      return undefined
    } catch (error) {
      if (error instanceof z.ZodError) {
        return error.issues[0]?.message
      }
      return undefined
    }
  }

  const validatePassword = (value: string) => {
    try {
      registrationSchema.shape.password.parse(value)
      return undefined
    } catch (error) {
      if (error instanceof z.ZodError) {
        return error.issues[0]?.message
      }
      return undefined
    }
  }

  const validateConfirmPassword = (value: string) => {
    const passwordValue = form.state.values.password
    if (!value) return 'Please confirm your password'
    if (value !== passwordValue) return "Passwords don't match"
    return undefined
  }

  return (
    <div
      className={cn('flex flex-col gap-6 w-full max-w-md', className)}
      {...props}
    >
      <div className="flex flex-col items-center gap-2 text-center">
        <h1 className="text-2xl font-bold">Create an account</h1>
        <p className="text-muted-foreground text-sm text-balance">
          Enter your information to get started
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
        {/* Name Fields Row */}
        <div className="grid grid-cols-2 gap-4">
          {/* First Name Field */}
          <div className="grid gap-2">
            <form.Field
              name="firstName"
              validators={{
                onChange: ({ value }) => validateFirstName(value),
              }}
              children={(field) => (
                <>
                  <Label htmlFor={field.name}>
                    First Name
                    <span className="text-destructive ml-1">*</span>
                  </Label>
                  <Input
                    id={field.name}
                    type="text"
                    placeholder="John"
                    value={field.state.value}
                    onBlur={field.handleBlur}
                    onChange={(e) => field.handleChange(e.target.value)}
                    className={cn(
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
                    autoComplete="given-name"
                  />
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

          {/* Last Name Field */}
          <div className="grid gap-2">
            <form.Field
              name="lastName"
              validators={{
                onChange: ({ value }) => validateLastName(value),
              }}
              children={(field) => (
                <>
                  <Label htmlFor={field.name}>
                    Last Name
                    <span className="text-destructive ml-1">*</span>
                  </Label>
                  <Input
                    id={field.name}
                    type="text"
                    placeholder="Doe"
                    value={field.state.value}
                    onBlur={field.handleBlur}
                    onChange={(e) => field.handleChange(e.target.value)}
                    className={cn(
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
                    autoComplete="family-name"
                  />
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
        </div>

        {/* Organization Name Field */}
        <div className="grid gap-2">
          <form.Field
            name="organizationName"
            validators={{
              onChange: ({ value }) => validateOrganizationName(value),
            }}
            children={(field) => (
              <>
                <Label htmlFor={field.name}>
                  Organization Name
                  <span className="text-destructive ml-1">*</span>
                </Label>
                <Input
                  id={field.name}
                  type="text"
                  placeholder="Acme Corporation"
                  value={field.state.value}
                  onBlur={field.handleBlur}
                  onChange={(e) => field.handleChange(e.target.value)}
                  className={cn(
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
                />
                <p className="text-xs text-muted-foreground">
                  Enter your organization or company name
                </p>
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

        {/* Email Field */}
        <div className="grid gap-2">
          <form.Field
            name="email"
            validators={{
              onChange: ({ value }) => validateEmail(value),
              onChangeAsyncDebounceMs: 500,
              onChangeAsync: async () => {
                // You can add async validation to check if email already exists
                // Example: const exists = await checkEmailExists(value)
                // if (exists) return 'Email already registered'
                await Promise.resolve()
                return undefined
              },
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
                    placeholder="john.doe@example.com"
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

        {/* Password Field */}
        <div className="grid gap-2">
          <form.Field
            name="password"
            validators={{
              onChange: ({ value }) => validatePassword(value),
            }}
            children={(field) => (
              <>
                <Label htmlFor={field.name}>
                  Password
                  <span className="text-destructive ml-1">*</span>
                </Label>
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
                    autoComplete="new-password"
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
                <p className="text-xs text-muted-foreground">
                  Must be at least 8 characters with uppercase, lowercase,
                  number and special character
                </p>
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

        {/* Confirm Password Field */}
        <div className="grid gap-2">
          <form.Field
            name="confirmPassword"
            validators={{
              onChange: ({ value }) => validateConfirmPassword(value),
            }}
            children={(field) => (
              <>
                <Label htmlFor={field.name}>
                  Confirm Password
                  <span className="text-destructive ml-1">*</span>
                </Label>
                <div className="relative">
                  <Input
                    id={field.name}
                    type={showConfirmPassword ? 'text' : 'password'}
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
                    autoComplete="new-password"
                  />
                  <button
                    type="button"
                    onClick={() => setShowConfirmPassword(!showConfirmPassword)}
                    className="absolute right-3 top-1/2 -translate-y-1/2 text-muted-foreground hover:text-foreground transition-colors"
                    aria-label={
                      showConfirmPassword ? 'Hide password' : 'Show password'
                    }
                  >
                    {showConfirmPassword ? (
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

        {/* Server Error Display */}
        {serverError && (
          <div className="bg-destructive/10 border border-destructive/20 rounded-md p-3 animate-in fade-in-50 slide-in-from-top-1">
            <p className="text-sm text-destructive flex items-center gap-2">
              <span className="inline-block w-1 h-1 bg-destructive rounded-full" />
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
              disabled={
                !canSubmit || isSubmitting || registerMutation.isPending
              }
            >
              {isSubmitting || registerMutation.isPending ? (
                <>
                  <Loader2Icon className="mr-2 h-4 w-4 animate-spin" />
                  Creating account...
                </>
              ) : (
                'Create account'
              )}
            </Button>
          )}
        />

        <p className="text-xs text-muted-foreground text-center">
          By creating an account, you agree to our{' '}
          <Link
            to="/"
            className="underline underline-offset-2 hover:text-foreground"
          >
            Terms of Service
          </Link>{' '}
          and{' '}
          <Link
            to="/"
            className="underline underline-offset-2 hover:text-foreground"
          >
            Privacy Policy
          </Link>
        </p>
      </form>

      {/* Sign in link */}
      <div className="text-center text-sm">
        <span className="text-muted-foreground">Already have an account? </span>
        <Link
          to="/login"
          className="font-medium text-primary underline-offset-4 hover:underline"
        >
          Sign in
        </Link>
      </div>
    </div>
  )
}
