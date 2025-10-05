import { Button } from '@/components/ui/button'
import { Card } from '@/components/ui/card'
import {
  IconChartLine,
  IconCode,
  IconPlus,
  IconRocket,
  IconWorld,
} from '@tabler/icons-react'

interface EmptyProjectsStateProps {
  onCreateProject: () => void
}

export function EmptyProjectsState({ onCreateProject }: EmptyProjectsStateProps) {
  return (
    <div className="flex flex-col items-center justify-center min-h-[600px] px-4">
      <div className="max-w-2xl mx-auto text-center space-y-6">
        {/* Icon */}
        <div className="relative">
          <div className="w-20 h-20 rounded-full bg-primary/10 flex items-center justify-center mx-auto">
            <IconChartLine className="w-10 h-10 text-primary" />
          </div>
          <div className="absolute -right-2 -top-2 w-8 h-8 rounded-full bg-green-100 dark:bg-green-900/30 flex items-center justify-center">
            <IconPlus className="w-4 h-4 text-green-600 dark:text-green-400" />
          </div>
        </div>

        {/* Title and description */}
        <div className="space-y-2">
          <h2 className="text-2xl font-bold">Create Your First Analytics Project</h2>
          <p className="text-muted-foreground max-w-md mx-auto">
            Start tracking your website's performance with Zori Analytics. Get insights about your visitors, page views, and user behavior.
          </p>
        </div>

        {/* Features */}
        <div className="grid grid-cols-1 md:grid-cols-3 gap-4 mt-8">
          <Card className="p-4 text-left">
            <div className="flex items-start gap-3">
              <div className="w-10 h-10 rounded-lg bg-blue-100 dark:bg-blue-900/30 flex items-center justify-center shrink-0">
                <IconRocket className="w-5 h-5 text-blue-600 dark:text-blue-400" />
              </div>
              <div>
                <h3 className="font-semibold text-sm mb-1">Quick Setup</h3>
                <p className="text-xs text-muted-foreground">
                  Add tracking to your site in under a minute
                </p>
              </div>
            </div>
          </Card>

          <Card className="p-4 text-left">
            <div className="flex items-start gap-3">
              <div className="w-10 h-10 rounded-lg bg-purple-100 dark:bg-purple-900/30 flex items-center justify-center shrink-0">
                <IconCode className="w-5 h-5 text-purple-600 dark:text-purple-400" />
              </div>
              <div>
                <h3 className="font-semibold text-sm mb-1">Simple Integration</h3>
                <p className="text-xs text-muted-foreground">
                  Just one script tag to add to your website
                </p>
              </div>
            </div>
          </Card>

          <Card className="p-4 text-left">
            <div className="flex items-start gap-3">
              <div className="w-10 h-10 rounded-lg bg-green-100 dark:bg-green-900/30 flex items-center justify-center shrink-0">
                <IconWorld className="w-5 h-5 text-green-600 dark:text-green-400" />
              </div>
              <div>
                <h3 className="font-semibold text-sm mb-1">Real-time Data</h3>
                <p className="text-xs text-muted-foreground">
                  See your analytics data within minutes
                </p>
              </div>
            </div>
          </Card>
        </div>

        {/* CTA */}
        <div className="pt-4">
          <Button onClick={onCreateProject} size="lg">
            <IconPlus className="mr-2 h-4 w-4" />
            Create Your First Project
          </Button>
          <p className="text-xs text-muted-foreground mt-4">
            No credit card required • Free tier available
          </p>
        </div>

        {/* Steps preview */}
        <Card className="p-6 mt-8 text-left bg-muted/30">
          <h3 className="font-semibold mb-4">How it works</h3>
          <ol className="space-y-3">
            <li className="flex items-start gap-3">
              <span className="w-6 h-6 rounded-full bg-primary text-primary-foreground text-xs flex items-center justify-center shrink-0 mt-0.5">
                1
              </span>
              <div>
                <p className="font-medium text-sm">Create a project</p>
                <p className="text-xs text-muted-foreground">Enter your project name and website URL</p>
              </div>
            </li>
            <li className="flex items-start gap-3">
              <span className="w-6 h-6 rounded-full bg-primary text-primary-foreground text-xs flex items-center justify-center shrink-0 mt-0.5">
                2
              </span>
              <div>
                <p className="font-medium text-sm">Add tracking script</p>
                <p className="text-xs text-muted-foreground">Copy and paste the script into your website's HTML</p>
              </div>
            </li>
            <li className="flex items-start gap-3">
              <span className="w-6 h-6 rounded-full bg-primary text-primary-foreground text-xs flex items-center justify-center shrink-0 mt-0.5">
                3
              </span>
              <div>
                <p className="font-medium text-sm">View analytics</p>
                <p className="text-xs text-muted-foreground">Watch your data flow in real-time</p>
              </div>
            </li>
          </ol>
        </Card>
      </div>
    </div>
  )
}
