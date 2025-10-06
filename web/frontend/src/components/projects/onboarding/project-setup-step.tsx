import { useState } from 'react'
import {
  IconBrandNextjs,
  IconBrandReact,
  IconBrandWordpress,
  IconCheck,
  IconCode,
  IconCopy,
  IconExternalLink,
  IconFileCode,
} from '@tabler/icons-react'
import { toast } from 'sonner'
import type { CreatedProject } from './project-onboarding'
import { Button } from '@/components/ui/button'
import { Label } from '@/components/ui/label'
import { Card } from '@/components/ui/card'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'

interface ProjectSetupStepProps {
  project: CreatedProject | null
}

export function ProjectSetupStep({ project }: ProjectSetupStepProps) {
  const [copiedScript, setCopiedScript] = useState(false)
  const [copiedToken, setCopiedToken] = useState(false)

  if (!project) {
    return (
      <div className="text-center py-8">
        <p className="text-muted-foreground">Loading project details...</p>
      </div>
    )
  }

  const handleCopyScript = (script: string) => {
    navigator.clipboard.writeText(script)
    setCopiedScript(true)
    toast.success('Script copied to clipboard!')
    setTimeout(() => setCopiedScript(false), 2000)
  }

  const handleCopyToken = () => {
    navigator.clipboard.writeText(project.projectToken)
    setCopiedToken(true)
    toast.success('Project token copied!')
    setTimeout(() => setCopiedToken(false), 2000)
  }

  const basicScript = `<!-- Zori Analytics -->
<script async src="https://analytics.zori.app/script.js" data-website-id="${project.projectToken}"></script>`

  const reactScript = `// Install the Zori Analytics React package
npm install @zori/analytics-react

// Add to your app
import { ZoriAnalytics } from '@zori/analytics-react'

function App() {
  return (
    <>
      <ZoriAnalytics websiteId="${project.projectToken}" />
      {/* Your app content */}
    </>
  )
}`

  const nextjsScript = `// pages/_app.js or app/layout.js
import Script from 'next/script'

export default function App({ Component, pageProps }) {
  return (
    <>
      <Script
        src="https://analytics.zori.app/script.js"
        data-website-id="${project.projectToken}"
        strategy="afterInteractive"
      />
      <Component {...pageProps} />
    </>
  )
}`

  const wordpressInstructions = `1. Go to your WordPress admin dashboard
2. Navigate to Appearance > Theme Editor
3. Open your theme's header.php file
4. Paste the script before the closing </head> tag
5. Save the changes

Or use a plugin like "Insert Headers and Footers"`

  return (
    <div className="space-y-6">
      <div className="text-center mb-8">
        <div className="w-16 h-16 rounded-full bg-green-100 dark:bg-green-900/30 flex items-center justify-center mx-auto mb-4">
          <IconCheck className="w-8 h-8 text-green-600 dark:text-green-400" />
        </div>
        <h2 className="text-2xl font-bold mb-2">
          Project Created Successfully!
        </h2>
        <p className="text-muted-foreground">
          Now let's add the tracking script to your website to start collecting
          analytics data.
        </p>
      </div>

      {/* Project Details */}
      <Card className="p-4 bg-muted/30">
        <div className="flex flex-col md:flex-row justify-evenly items-center gap-4 text-sm">
          <div className="flex flex-col items-center text-center">
            <span className="text-muted-foreground">Project Name:</span>
            <p className="font-medium">{project.name}</p>
          </div>
          <div className="flex flex-col items-center text-center">
            <span className="text-muted-foreground">Website:</span>
            <p className="font-medium">{project.domain}</p>
          </div>
          <div className="flex flex-col items-center text-center">
            <span className="text-muted-foreground">Project Token:</span>
            <div className="flex items-center gap-2">
              <code className="text-xs bg-background px-2  overflow-clip py-1 rounded border">
                {project.projectToken}
              </code>
              <Button
                variant="ghost"
                size="icon"
                className="h-6 w-6"
                onClick={handleCopyToken}
              >
                {copiedToken ? (
                  <IconCheck className="h-3 w-3 text-green-500" />
                ) : (
                  <IconCopy className="h-3 w-3" />
                )}
              </Button>
            </div>
          </div>
        </div>
      </Card>

      {/* Installation Methods */}
      <div>
        <h3 className="text-lg font-semibold mb-4">
          Choose your installation method
        </h3>

        <Tabs defaultValue="html" className="w-full">
          <TabsList className="grid w-full grid-cols-3">
            <TabsTrigger value="html" className="flex items-center gap-2">
              <IconFileCode className="w-4 h-4" />
              HTML
            </TabsTrigger>
            <TabsTrigger value="react" className="flex items-center gap-2">
              <IconBrandReact className="w-4 h-4" />
              React
            </TabsTrigger>
            <TabsTrigger value="nextjs" className="flex items-center gap-2">
              <IconBrandNextjs className="w-4 h-4" />
              Next.js
            </TabsTrigger>
          </TabsList>

          <TabsContent value="html" className="space-y-4">
            <div>
              <div className="flex items-center justify-between mb-2">
                <Label>HTML Script Tag</Label>
                <Button
                  variant="outline"
                  size="sm"
                  onClick={() => handleCopyScript(basicScript)}
                  className="flex items-center gap-2"
                >
                  {copiedScript ? (
                    <IconCheck className="w-4 h-4" />
                  ) : (
                    <IconCopy className="w-4 h-4" />
                  )}
                  Copy
                </Button>
              </div>
              <div className="rounded-lg border bg-muted/50 p-4">
                <code className="text-sm whitespace-pre-wrap">
                  {basicScript}
                </code>
              </div>
              <div className="mt-4 space-y-2">
                <p className="text-sm text-muted-foreground">
                  <strong>Instructions:</strong>
                </p>
                <ol className="text-sm text-muted-foreground space-y-1 list-decimal list-inside">
                  <li>Copy the script above</li>
                  <li>
                    Paste it in the{' '}
                    <code className="bg-muted px-1 rounded">&lt;head&gt;</code>{' '}
                    section of your HTML
                  </li>
                  <li>
                    Place it before the closing{' '}
                    <code className="bg-muted px-1 rounded">&lt;/head&gt;</code>{' '}
                    tag
                  </li>
                  <li>Deploy your changes</li>
                </ol>
              </div>
            </div>
          </TabsContent>

          <TabsContent value="react" className="space-y-4">
            <div>
              <div className="flex items-center justify-between mb-2">
                <Label>React Integration</Label>
                <Button
                  variant="outline"
                  size="sm"
                  onClick={() => handleCopyScript(reactScript)}
                  className="flex items-center gap-2"
                >
                  {copiedScript ? (
                    <IconCheck className="w-4 h-4" />
                  ) : (
                    <IconCopy className="w-4 h-4" />
                  )}
                  Copy
                </Button>
              </div>
              <div className="rounded-lg border bg-muted/50 p-4">
                <code className="text-sm whitespace-pre-wrap">
                  {reactScript}
                </code>
              </div>
            </div>
          </TabsContent>

          <TabsContent value="nextjs" className="space-y-4">
            <div>
              <div className="flex items-center justify-between mb-2">
                <Label>Next.js Integration</Label>
                <Button
                  variant="outline"
                  size="sm"
                  onClick={() => handleCopyScript(nextjsScript)}
                  className="flex items-center gap-2"
                >
                  {copiedScript ? (
                    <IconCheck className="w-4 h-4" />
                  ) : (
                    <IconCopy className="w-4 h-4" />
                  )}
                  Copy
                </Button>
              </div>
              <div className="rounded-lg border bg-muted/50 p-4">
                <code className="text-sm whitespace-pre-wrap">
                  {nextjsScript}
                </code>
              </div>
            </div>
          </TabsContent>
        </Tabs>
      </div>

      {/* Help Section */}
      <Card className="p-4 bg-blue-50 dark:bg-blue-950/30 border border-blue-200 dark:border-blue-800">
        <div className="flex gap-3">
          <IconCode className="h-5 w-5 text-blue-600 dark:text-blue-400 mt-0.5 shrink-0" />
          <div className="space-y-2">
            <h4 className="font-medium text-blue-900 dark:text-blue-100">
              Need help with installation?
            </h4>
            <p className="text-sm text-blue-800 dark:text-blue-200">
              Check out our detailed installation guide or contact our support
              team for assistance.
            </p>
            <div className="flex gap-2">
              <Button
                variant="outline"
                size="sm"
                className="text-blue-700 border-blue-300"
              >
                <IconExternalLink className="w-4 h-4 mr-2" />
                View Docs
              </Button>
              <Button
                variant="outline"
                size="sm"
                className="text-blue-700 border-blue-300"
              >
                Contact Support
              </Button>
            </div>
          </div>
        </div>
      </Card>
    </div>
  )
}
