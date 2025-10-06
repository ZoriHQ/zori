import {
  IconChartLine,
  IconCheck,
  IconClock,
  IconExternalLink,
  IconEye,
  IconRefresh,
  IconUsers,
} from '@tabler/icons-react'
import type { CreatedProject } from './project-onboarding'
import { Button } from '@/components/ui/button'
import { Card } from '@/components/ui/card'

interface ProjectCompleteStepProps {
  project: CreatedProject | null
}

export function ProjectCompleteStep({ project }: ProjectCompleteStepProps) {
  if (!project) {
    return (
      <div className="text-center py-8">
        <p className="text-muted-foreground">Loading project details...</p>
      </div>
    )
  }

  return (
    <div className="space-y-6">
      <div className="text-center mb-8">
        <div className="w-20 h-20 rounded-full bg-green-100 dark:bg-green-900/30 flex items-center justify-center mx-auto mb-4">
          <IconCheck className="w-10 h-10 text-green-600 dark:text-green-400" />
        </div>
        <h2 className="text-2xl font-bold mb-2">🎉 You're all set!</h2>
        <p className="text-muted-foreground max-w-md mx-auto">
          Your analytics project <strong>{project.name}</strong> has been
          created and is ready to start collecting data.
        </p>
      </div>

      {/* Project Summary */}
      <Card className="p-6 bg-gradient-to-r from-green-50 to-blue-50 dark:from-green-950/30 dark:to-blue-950/30 border-green-200 dark:border-green-800">
        <div className="text-center space-y-4">
          <h3 className="text-lg font-semibold">
            Project Successfully Created
          </h3>
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4 text-sm">
            <div className="space-y-2">
              <p className="text-muted-foreground">Project Name</p>
              <p className="font-medium">{project.name}</p>
            </div>
            <div className="space-y-2">
              <p className="text-muted-foreground">Website</p>
              <p className="font-medium">{project.domain}</p>
            </div>
          </div>
        </div>
      </Card>

      {/* What's Next */}
      <div className="space-y-4">
        <h3 className="text-lg font-semibold">What happens next?</h3>

        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          <Card className="p-4">
            <div className="flex items-start gap-3">
              <div className="w-8 h-8 rounded-full bg-blue-100 dark:bg-blue-900/30 flex items-center justify-center shrink-0">
                <IconClock className="w-4 h-4 text-blue-600 dark:text-blue-400" />
              </div>
              <div>
                <h4 className="font-medium mb-1">Data Collection Starts</h4>
                <p className="text-sm text-muted-foreground">
                  Once you deploy the tracking script, data will start appearing
                  within 5-10 minutes.
                </p>
              </div>
            </div>
          </Card>

          <Card className="p-4">
            <div className="flex items-start gap-3">
              <div className="w-8 h-8 rounded-full bg-purple-100 dark:bg-purple-900/30 flex items-center justify-center shrink-0">
                <IconChartLine className="w-4 h-4 text-purple-600 dark:text-purple-400" />
              </div>
              <div>
                <h4 className="font-medium mb-1">Real-time Analytics</h4>
                <p className="text-sm text-muted-foreground">
                  View live visitor data, page views, and user behavior on your
                  dashboard.
                </p>
              </div>
            </div>
          </Card>
        </div>
      </div>

      {/* Analytics Preview */}
      <div className="space-y-4">
        <h3 className="text-lg font-semibold">Analytics you'll get</h3>

        <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
          <Card className="p-4 text-center">
            <div className="w-12 h-12 rounded-lg bg-blue-100 dark:bg-blue-900/30 flex items-center justify-center mx-auto mb-3">
              <IconUsers className="w-6 h-6 text-blue-600 dark:text-blue-400" />
            </div>
            <h4 className="font-semibold mb-1">Visitor Insights</h4>
            <p className="text-sm text-muted-foreground">
              Track unique visitors, returning users, and user sessions
            </p>
          </Card>

          <Card className="p-4 text-center">
            <div className="w-12 h-12 rounded-lg bg-green-100 dark:bg-green-900/30 flex items-center justify-center mx-auto mb-3">
              <IconEye className="w-6 h-6 text-green-600 dark:text-green-400" />
            </div>
            <h4 className="font-semibold mb-1">Page Analytics</h4>
            <p className="text-sm text-muted-foreground">
              Monitor page views, bounce rates, and popular content
            </p>
          </Card>

          <Card className="p-4 text-center">
            <div className="w-12 h-12 rounded-lg bg-purple-100 dark:bg-purple-900/30 flex items-center justify-center mx-auto mb-3">
              <IconChartLine className="w-6 h-6 text-purple-600 dark:text-purple-400" />
            </div>
            <h4 className="font-semibold mb-1">Performance Metrics</h4>
            <p className="text-sm text-muted-foreground">
              Analyze traffic sources, device types, and geographic data
            </p>
          </Card>
        </div>
      </div>

      {/* Action Buttons */}
      <div className="flex flex-col sm:flex-row gap-3 pt-4">
        <Button className="flex-1" size="lg">
          <IconChartLine className="w-4 h-4 mr-2" />
          View Analytics Dashboard
        </Button>
        <Button variant="outline" className="flex-1" size="lg">
          <IconExternalLink className="w-4 h-4 mr-2" />
          Test Your Installation
        </Button>
      </div>

      {/* Help Section */}
      <Card className="p-4 bg-muted/30">
        <div className="flex items-start gap-3">
          <div className="w-8 h-8 rounded-full bg-yellow-100 dark:bg-yellow-900/30 flex items-center justify-center shrink-0">
            <IconRefresh className="w-4 h-4 text-yellow-600 dark:text-yellow-400" />
          </div>
          <div className="space-y-2">
            <h4 className="font-medium">Don't see data yet?</h4>
            <p className="text-sm text-muted-foreground">
              It can take a few minutes for data to appear. Make sure the
              tracking script is properly installed and try visiting your
              website.
            </p>
            <div className="flex gap-2">
              <Button variant="outline" size="sm">
                Troubleshooting Guide
              </Button>
              <Button variant="outline" size="sm">
                Contact Support
              </Button>
            </div>
          </div>
        </div>
      </Card>

      {/* Success Message */}
      <div className="text-center pt-4 pb-2">
        <p className="text-sm text-muted-foreground">
          Welcome to Zori Analytics! 🚀
        </p>
      </div>
    </div>
  )
}
