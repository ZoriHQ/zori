'use client'

import {
  forwardRef,
  type HTMLAttributes,
  useCallback,
  useEffect,
  useRef,
  useState,
} from 'react'

import {
  ChevronDownIcon,
  ChevronsUpDown,
  SettingsIcon,
  type SVGProps,
} from 'lucide-react'
import {
  Breadcrumb,
  BreadcrumbItem,
  BreadcrumbList,
  BreadcrumbSeparator,
} from './ui/breadcrumb'
import { Button } from './ui/button'
import {
  NavigationMenu,
  NavigationMenuItem,
  NavigationMenuLink,
  NavigationMenuList,
} from './ui/navigation-menu'
import { Popover, PopoverContent, PopoverTrigger } from './ui/popover'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from './ui/select'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from './ui/dropdown-menu'
import { Avatar, AvatarFallback, AvatarImage } from './ui/avatar'
import { cn } from '@/lib/utils'

// Hamburger icon component
const HamburgerIcon = ({ className, ...props }: SVGProps<SVGElement>) => (
  <svg
    className={cn('pointer-events-none', className)}
    width={16}
    height={16}
    viewBox="0 0 24 24"
    fill="none"
    stroke="currentColor"
    strokeWidth="2"
    strokeLinecap="round"
    strokeLinejoin="round"
    xmlns="http://www.w3.org/2000/svg"
    {...props}
  >
    <path
      d="M4 12L20 12"
      className="origin-center -translate-y-[7px] transition-all duration-300 ease-[cubic-bezier(.5,.85,.25,1.1)] group-aria-expanded:translate-x-0 group-aria-expanded:translate-y-0 group-aria-expanded:rotate-[315deg]"
    />
    <path
      d="M4 12H20"
      className="origin-center transition-all duration-300 ease-[cubic-bezier(.5,.85,.25,1.8)] group-aria-expanded:rotate-45"
    />
    <path
      d="M4 12H20"
      className="origin-center translate-y-[7px] transition-all duration-300 ease-[cubic-bezier(.5,.85,.25,1.1)] group-aria-expanded:translate-y-0 group-aria-expanded:rotate-[135deg]"
    />
  </svg>
)

// Settings Menu Component
const SettingsMenu = ({
  onItemClick,
}: {
  onItemClick?: (item: string) => void
}) => (
  <DropdownMenu>
    <DropdownMenuTrigger asChild>
      <Button variant="ghost" size="icon" className="h-9 w-9">
        <SettingsIcon className="h-4 w-4" />
        <span className="sr-only">Settings</span>
      </Button>
    </DropdownMenuTrigger>
    <DropdownMenuContent align="end" className="w-56">
      <DropdownMenuLabel>Settings</DropdownMenuLabel>
      <DropdownMenuSeparator />
      <DropdownMenuItem onClick={() => onItemClick?.('preferences')}>
        Preferences
      </DropdownMenuItem>
      <DropdownMenuItem onClick={() => onItemClick?.('billing')}>
        Billing
      </DropdownMenuItem>
      <DropdownMenuItem onClick={() => onItemClick?.('team')}>
        Team Settings
      </DropdownMenuItem>
      <DropdownMenuItem onClick={() => onItemClick?.('integrations')}>
        Integrations
      </DropdownMenuItem>
      <DropdownMenuSeparator />
      <DropdownMenuItem onClick={() => onItemClick?.('api-keys')}>
        API Keys
      </DropdownMenuItem>
    </DropdownMenuContent>
  </DropdownMenu>
)

// User Menu Component
const UserMenu = ({
  userName = 'John Doe',
  userEmail = 'john@example.com',
  userAvatar,
  onItemClick,
}: {
  userName?: string
  userEmail?: string
  userAvatar?: string
  onItemClick?: (item: string) => void
}) => (
  <DropdownMenu>
    <DropdownMenuTrigger asChild>
      <Button
        variant="ghost"
        className="h-9 px-2 py-0 hover:bg-accent hover:text-accent-foreground"
      >
        <Avatar className="h-7 w-7">
          <AvatarImage src={userAvatar} alt={userName} />
          <AvatarFallback className="text-xs">
            {userName
              .split(' ')
              .map((n) => n[0])
              .join('')}
          </AvatarFallback>
        </Avatar>
        <ChevronDownIcon className="h-3 w-3 ml-1" />
        <span className="sr-only">User menu</span>
      </Button>
    </DropdownMenuTrigger>
    <DropdownMenuContent align="end" className="w-56">
      <DropdownMenuLabel>
        <div className="flex flex-col space-y-1">
          <p className="text-sm font-medium leading-none">{userName}</p>
          <p className="text-xs leading-none text-muted-foreground">
            {userEmail}
          </p>
        </div>
      </DropdownMenuLabel>
      <DropdownMenuSeparator />
      <DropdownMenuItem onClick={() => onItemClick?.('profile')}>
        Profile
      </DropdownMenuItem>
      <DropdownMenuItem onClick={() => onItemClick?.('account')}>
        Account
      </DropdownMenuItem>
      <DropdownMenuItem onClick={() => onItemClick?.('support')}>
        Support
      </DropdownMenuItem>
      <DropdownMenuSeparator />
      <DropdownMenuItem onClick={() => onItemClick?.('logout')}>
        Log out
      </DropdownMenuItem>
    </DropdownMenuContent>
  </DropdownMenu>
)

// Types
export interface NavbarNavItem {
  href?: string
  label: string
}

export interface NavbarAccountType {
  value: string
  label: string
}

export interface NavbarProject {
  value: string
  label: string
}

export interface NavbarProps extends HTMLAttributes<HTMLElement> {
  navigationLinks?: Array<NavbarNavItem>
  accountTypes?: Array<NavbarAccountType>
  defaultAccountType?: string
  projects?: Array<NavbarProject>
  defaultProject?: string
  userName?: string
  userEmail?: string
  userAvatar?: string
  onNavItemClick?: (href: string) => void
  onAccountTypeChange?: (accountType: string) => void
  onProjectChange?: (project: string) => void
  onSettingsItemClick?: (item: string) => void
  onUserItemClick?: (item: string) => void
}

// Default navigation links
const defaultNavigationLinks: Array<NavbarNavItem> = [
  { href: '#', label: 'Dashboard' },
  { href: '#', label: 'Docs' },
  { href: '#', label: 'API reference' },
]

// Default account types
const defaultAccountTypes: Array<NavbarAccountType> = [
  { value: 'personal', label: 'Personal' },
  { value: 'team', label: 'Team' },
  { value: 'business', label: 'Business' },
]

// Default projects
const defaultProjects: Array<NavbarProject> = [
  { value: '1', label: 'Main project' },
  { value: '2', label: 'Origin project' },
]

export const Navbar = forwardRef<HTMLElement, NavbarProps>(
  (
    {
      navigationLinks = defaultNavigationLinks,
      accountTypes = defaultAccountTypes,
      defaultAccountType = 'personal',
      projects = defaultProjects,
      defaultProject = '1',
      userName = 'John Doe',
      userEmail = 'john@example.com',
      userAvatar,
      onNavItemClick,
      onAccountTypeChange,
      onProjectChange,
      onSettingsItemClick,
      onUserItemClick,
      ...props
    },
    ref,
  ) => {
    const [isMobile, setIsMobile] = useState(false)
    const containerRef = useRef<HTMLElement>(null)

    useEffect(() => {
      const checkWidth = () => {
        if (containerRef.current) {
          const width = containerRef.current.offsetWidth
          setIsMobile(width < 768) // 768px is md breakpoint
        }
      }

      checkWidth()

      const resizeObserver = new ResizeObserver(checkWidth)
      if (containerRef.current) {
        resizeObserver.observe(containerRef.current)
      }

      return () => {
        resizeObserver.disconnect()
      }
    }, [])

    // Combine refs
    const combinedRef = useCallback(
      (node: HTMLElement | null) => {
        containerRef.current = node
        if (typeof ref === 'function') {
          ref(node)
        } else if (ref) {
          ref.current = node
        }
      },
      [ref],
    )

    return (
      <header
        ref={combinedRef}
        className={cn(
          'sticky top-0 z-50 w-full border-b bg-background/95 backdrop-blur supports-[backdrop-filter]:bg-background/60 px-4 md:px-6 [&_*]:no-underline',
        )}
        {...props}
      >
        <div className="container mx-auto flex h-16 max-w-screen-2xl items-center justify-between gap-2">
          {/* Left side */}
          <div className="flex items-center gap-2 min-w-0 flex-1">
            {/* Mobile menu trigger */}
            {isMobile && (
              <Popover>
                <PopoverTrigger asChild>
                  <Button
                    className="group h-8 w-8 hover:bg-accent hover:text-accent-foreground shrink-0"
                    variant="ghost"
                    size="icon"
                  >
                    <HamburgerIcon />
                  </Button>
                </PopoverTrigger>
                <PopoverContent align="start" className="w-80 p-2">
                  <div className="space-y-4">
                    {/* Context switchers in mobile menu */}
                    <div className="space-y-2">
                      <div className="text-xs font-medium text-muted-foreground uppercase tracking-wide">
                        Context
                      </div>
                      <div className="space-y-2">
                        <Select
                          defaultValue={defaultAccountType}
                          onValueChange={onAccountTypeChange}
                        >
                          <SelectTrigger className="h-9">
                            <SelectValue placeholder="Select account type" />
                          </SelectTrigger>
                          <SelectContent>
                            {accountTypes.map((accountType) => (
                              <SelectItem
                                key={accountType.value}
                                value={accountType.value}
                              >
                                {accountType.label}
                              </SelectItem>
                            ))}
                          </SelectContent>
                        </Select>
                        <Select
                          defaultValue={defaultProject}
                          onValueChange={onProjectChange}
                        >
                          <SelectTrigger className="h-9">
                            <SelectValue placeholder="Select project" />
                          </SelectTrigger>
                          <SelectContent>
                            {projects.map((project) => (
                              <SelectItem
                                key={project.value}
                                value={project.value}
                              >
                                {project.label}
                              </SelectItem>
                            ))}
                          </SelectContent>
                        </Select>
                      </div>
                    </div>
                    {/* Navigation links in mobile menu */}
                    <div className="space-y-2">
                      <div className="text-xs font-medium text-muted-foreground uppercase tracking-wide">
                        Navigation
                      </div>
                      <NavigationMenu className="max-w-none">
                        <NavigationMenuList className="flex-col items-start gap-0 w-full">
                          {navigationLinks.map((link, index) => (
                            <NavigationMenuItem key={index} className="w-full">
                              <button
                                onClick={(e) => {
                                  e.preventDefault()
                                  if (onNavItemClick && link.href)
                                    onNavItemClick(link.href)
                                }}
                                className="flex w-full items-center rounded-md px-3 py-2 text-sm font-medium transition-colors hover:bg-accent hover:text-accent-foreground focus:bg-accent focus:text-accent-foreground cursor-pointer no-underline"
                              >
                                {link.label}
                              </button>
                            </NavigationMenuItem>
                          ))}
                        </NavigationMenuList>
                      </NavigationMenu>
                    </div>
                  </div>
                </PopoverContent>
              </Popover>
            )}
            {/* Breadcrumb - Hidden on mobile */}
            {!isMobile && (
              <Breadcrumb className="min-w-0">
                <BreadcrumbList className="flex-wrap">
                  <BreadcrumbItem>
                    <Select
                      defaultValue={defaultAccountType}
                      onValueChange={onAccountTypeChange}
                    >
                      <SelectTrigger className="focus-visible:bg-accent text-foreground h-8 p-1.5 focus-visible:ring-0 border-none shadow-none bg-transparent hover:bg-accent max-w-[120px]">
                        <SelectValue placeholder="Account" />
                        <ChevronsUpDown
                          size={14}
                          className="text-muted-foreground/80 ml-1 shrink-0"
                        />
                      </SelectTrigger>
                      <SelectContent className="[&_*[role=option]]:ps-2 [&_*[role=option]]:pe-8 [&_*[role=option]>span]:start-auto [&_*[role=option]>span]:end-2">
                        {accountTypes.map((accountType) => (
                          <SelectItem
                            key={accountType.value}
                            value={accountType.value}
                          >
                            {accountType.label}
                          </SelectItem>
                        ))}
                      </SelectContent>
                    </Select>
                  </BreadcrumbItem>
                  <BreadcrumbSeparator className="shrink-0">
                    {' '}
                    /{' '}
                  </BreadcrumbSeparator>
                  <BreadcrumbItem>
                    <Select
                      defaultValue={defaultProject}
                      onValueChange={onProjectChange}
                    >
                      <SelectTrigger className="focus-visible:bg-accent text-foreground h-8 p-1.5 focus-visible:ring-0 border-none shadow-none bg-transparent hover:bg-accent max-w-[140px]">
                        <SelectValue placeholder="Project" />
                        <ChevronsUpDown
                          size={14}
                          className="text-muted-foreground/80 ml-1 shrink-0"
                        />
                      </SelectTrigger>
                      <SelectContent className="[&_*[role=option]]:ps-2 [&_*[role=option]]:pe-8 [&_*[role=option]>span]:start-auto [&_*[role=option]>span]:end-2">
                        {projects.map((project) => (
                          <SelectItem key={project.value} value={project.value}>
                            {project.label}
                          </SelectItem>
                        ))}
                      </SelectContent>
                    </Select>
                  </BreadcrumbItem>
                </BreadcrumbList>
              </Breadcrumb>
            )}
          </div>
          {/* Right side */}
          <div className="flex items-center gap-2 shrink-0">
            {/* Nav menu - Desktop only */}
            {!isMobile && (
              <NavigationMenu className="hidden lg:flex">
                <NavigationMenuList className="gap-1">
                  {navigationLinks.map((link, index) => (
                    <NavigationMenuItem key={index}>
                      <NavigationMenuLink
                        href={link.href}
                        onClick={(e) => {
                          e.preventDefault()
                          if (onNavItemClick && link.href)
                            onNavItemClick(link.href)
                        }}
                        className="text-muted-foreground hover:text-primary py-1.5 font-medium transition-colors cursor-pointer group inline-flex h-9 w-max items-center justify-center rounded-md bg-background px-3 text-sm focus:bg-accent focus:text-accent-foreground focus:outline-none disabled:pointer-events-none disabled:opacity-50"
                      >
                        {link.label}
                      </NavigationMenuLink>
                    </NavigationMenuItem>
                  ))}
                </NavigationMenuList>
              </NavigationMenu>
            )}
            {/* Settings - Hidden on small mobile */}
            <div className="hidden sm:flex">
              <SettingsMenu onItemClick={onSettingsItemClick} />
            </div>
            {/* User menu */}
            <UserMenu
              userName={userName}
              userEmail={userEmail}
              userAvatar={userAvatar}
              onItemClick={onUserItemClick}
            />
          </div>
        </div>
      </header>
    )
  },
)

Navbar.displayName = 'Navbar'

export { HamburgerIcon, SettingsMenu, UserMenu }
