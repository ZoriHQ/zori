import {
  IconCode,
  IconRocket,
  IconShield,
  IconWorld,
} from '@tabler/icons-react'
import type { ProjectFormData } from './project-onboarding'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Checkbox } from '@/components/ui/checkbox'
import { Card } from '@/components/ui/card'

interface ProjectDetailsStepProps {
  data: ProjectFormData
  onChange: (data: ProjectFormData) => void
}

export function ProjectDetailsStep({
  data,
  onChange,
}: ProjectDetailsStepProps) {
  return (
    <div className="space-y-6">
      <div className="text-center mb-8">
        <h2 className="text-2xl font-bold mb-2">Let's set up your project</h2>
        <p className="text-muted-foreground">
          Tell us about your website so we can configure analytics tracking for
          you.
        </p>
      </div>

      {/* Form */}
      <div className="max-w-lg mx-auto space-y-6">
        <div className="space-y-2">
          <Label htmlFor="project-name">Project Name *</Label>
          <Input
            id="project-name"
            placeholder="My Awesome Website"
            value={data.name}
            onChange={(e) => onChange({ ...data, name: e.target.value })}
            className="h-12"
          />
          <p className="text-sm text-muted-foreground">
            A friendly name to identify your project in the dashboard
          </p>
        </div>

        <div className="space-y-2">
          <Label htmlFor="website-url">Website URL *</Label>
          <div className="relative">
            <div className="absolute left-3 top-1/2 transform -translate-y-1/2">
              <IconWorld className="h-4 w-4 text-muted-foreground" />
            </div>
            <Input
              id="website-url"
              type="url"
              placeholder="https://example.com"
              value={data.websiteUrl}
              onChange={(e) =>
                onChange({ ...data, websiteUrl: e.target.value })
              }
              className="h-12 pl-10"
            />
          </div>
          <p className="text-sm text-muted-foreground">
            The full URL of the website you want to track
          </p>
        </div>

        <div className="flex items-start space-x-3 p-4 border rounded-lg">
          <Checkbox
            id="allow-localhost"
            checked={data.allowLocalhost}
            onCheckedChange={(checked) =>
              onChange({ ...data, allowLocalhost: checked as boolean })
            }
          />
          <div className="space-y-1">
            <Label
              htmlFor="allow-localhost"
              className="text-sm font-medium cursor-pointer"
            >
              Allow tracking on localhost
            </Label>
            <p className="text-xs text-muted-foreground">
              Enable this if you want to test analytics on your local
              development environment
            </p>
          </div>
        </div>
      </div>

      {/* Features Preview */}
      <div className="mt-12">
        <h3 className="text-lg font-semibold text-center mb-6">
          What you'll get
        </h3>
        <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
          <Card className="p-4 text-center">
            <div className="w-12 h-12 rounded-lg bg-blue-100 dark:bg-blue-900/30 flex items-center justify-center mx-auto mb-3">
              <IconRocket className="w-6 h-6 text-blue-600 dark:text-blue-400" />
            </div>
            <h4 className="font-semibold mb-2">Quick Setup</h4>
            <p className="text-sm text-muted-foreground">
              Get your analytics up and running in under 2 minutes with our
              simple integration
            </p>
          </Card>

          <Card className="p-4 text-center">
            <div className="w-12 h-12 rounded-lg bg-purple-100 dark:bg-purple-900/30 flex items-center justify-center mx-auto mb-3">
              <IconCode className="w-6 h-6 text-purple-600 dark:text-purple-400" />
            </div>
            <h4 className="font-semibold mb-2">Simple Integration</h4>
            <p className="text-sm text-muted-foreground">
              Just one lightweight script tag to add to your website - no
              complex setup required
            </p>
          </Card>

          <Card className="p-4 text-center">
            <div className="w-12 h-12 rounded-lg bg-green-100 dark:bg-green-900/30 flex items-center justify-center mx-auto mb-3">
              <IconShield className="w-6 h-6 text-green-600 dark:text-green-400" />
            </div>
            <h4 className="font-semibold mb-2">Privacy-First</h4>
            <p className="text-sm text-muted-foreground">
              GDPR compliant analytics that respects your visitors' privacy by
              design
            </p>
          </Card>
        </div>
      </div>
    </div>
  )
}
