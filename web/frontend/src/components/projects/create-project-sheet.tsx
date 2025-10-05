import { useState } from 'react'
import { Button } from '@/components/ui/button'
import { Checkbox } from '@/components/ui/checkbox'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetFooter,
  SheetHeader,
  SheetTitle,
  SheetTrigger,
} from '@/components/ui/sheet'
import {
  IconCode,
  IconCopy,
  IconPlus,
  IconWorld,
} from '@tabler/icons-react'
import { toast } from 'sonner'
import { useCreateProject } from '@/hooks/use-projects'

interface CreateProjectSheetProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  trigger?: React.ReactNode
}

export function CreateProjectSheet({
  open,
  onOpenChange,
  trigger,
}: CreateProjectSheetProps) {
  const [createStep, setCreateStep] = useState<'form' | 'script'>('form')
  const [newProject, setNewProject] = useState({
    name: '',
    websiteUrl: '',
    allowLocalhost: false,
  })
  const [createdProject, setCreatedProject] = useState<{
    id: string
    projectToken: string
    name: string
    domain: string
  } | null>(null)

  const createProjectMutation = useCreateProject()

  const handleCreateProject = () => {
    createProjectMutation.mutate(
      {
        name: newProject.name,
        website_url: newProject.websiteUrl,
        allow_localhost: newProject.allowLocalhost,
      },
      {
        onSuccess: (data) => {
          setCreatedProject({
            id: data.id,
            projectToken: data.project_token,
            name: data.name,
            domain: data.domain,
          })
          setCreateStep('script')
          toast.success('Project created successfully!')
        },
        onError: (error) => {
          toast.error('Failed to create project: ' + error.message)
        },
      },
    )
  }

  const handleCopyScript = () => {
    const script = `<!-- Zori Analytics -->
<script async src="https://analytics.zori.app/script.js" data-website-id="${createdProject?.projectToken}"></script>`
    navigator.clipboard.writeText(script)
    toast.success('Script copied to clipboard!')
  }

  const handleCopyToken = () => {
    if (createdProject?.projectToken) {
      navigator.clipboard.writeText(createdProject.projectToken)
      toast.success('Project token copied to clipboard!')
    }
  }

  const resetForm = () => {
    setNewProject({ name: '', websiteUrl: '', allowLocalhost: false })
    setCreatedProject(null)
    setCreateStep('form')
  }

  const handleOpenChange = (open: boolean) => {
    onOpenChange(open)
    if (!open) {
      resetForm()
    }
  }

  return (
    <Sheet open={open} onOpenChange={handleOpenChange}>
      {trigger && <SheetTrigger asChild>{trigger}</SheetTrigger>}
      <SheetContent className="sm:max-w-[525px]">
        {createStep === 'form' ? (
          <>
            <SheetHeader>
              <SheetTitle>Create Analytics Project</SheetTitle>
              <SheetDescription>
                Set up tracking for a new website. You'll get a script to add to your site.
              </SheetDescription>
            </SheetHeader>
            <div className="grid gap-4 py-4">
              <div className="grid gap-2">
                <Label htmlFor="name">Project Name</Label>
                <Input
                  id="name"
                  placeholder="My Awesome Website"
                  value={newProject.name}
                  onChange={(e) =>
                    setNewProject({ ...newProject, name: e.target.value })
                  }
                />
                <p className="text-sm text-muted-foreground">
                  A friendly name to identify your project
                </p>
              </div>
              <div className="grid gap-2">
                <Label htmlFor="website">Project Website</Label>
                <div className="flex gap-2">
                  <div className="flex items-center justify-center px-3 bg-muted rounded-md">
                    <IconWorld className="h-4 w-4 text-muted-foreground" />
                  </div>
                  <Input
                    id="website"
                    type="url"
                    placeholder="https://example.com"
                    value={newProject.websiteUrl}
                    onChange={(e) =>
                      setNewProject({
                        ...newProject,
                        websiteUrl: e.target.value,
                      })
                    }
                  />
                </div>
                <p className="text-sm text-muted-foreground">
                  The URL of the website you want to track
                </p>
              </div>
              <div className="flex items-center space-x-2">
                <Checkbox
                  id="localhost"
                  checked={newProject.allowLocalhost}
                  onCheckedChange={(checked) =>
                    setNewProject({
                      ...newProject,
                      allowLocalhost: checked as boolean,
                    })
                  }
                />
                <Label
                  htmlFor="localhost"
                  className="text-sm font-normal cursor-pointer"
                >
                  Allow tracking on localhost (for development)
                </Label>
              </div>
            </div>
            <SheetFooter>
              <Button
                variant="outline"
                onClick={() => handleOpenChange(false)}
              >
                Cancel
              </Button>
              <Button
                onClick={handleCreateProject}
                disabled={
                  !newProject.name ||
                  !newProject.websiteUrl ||
                  createProjectMutation.isPending
                }
              >
                {createProjectMutation.isPending
                  ? 'Creating...'
                  : 'Create Project'}
              </Button>
            </SheetFooter>
          </>
        ) : (
          <>
            <SheetHeader>
              <SheetTitle>Project Created Successfully!</SheetTitle>
              <SheetDescription>
                Add this tracking script to your website to start collecting analytics data.
              </SheetDescription>
            </SheetHeader>
            <div className="grid gap-4 py-4">
              <div className="grid gap-2">
                <Label>Project Details</Label>
                <div className="rounded-lg border p-3 space-y-2">
                  <div className="flex justify-between text-sm">
                    <span className="text-muted-foreground">Name:</span>
                    <span className="font-medium">
                      {createdProject?.name}
                    </span>
                  </div>
                  <div className="flex justify-between text-sm">
                    <span className="text-muted-foreground">
                      Website:
                    </span>
                    <span className="font-medium">
                      {createdProject?.domain}
                    </span>
                  </div>
                  <div className="flex justify-between text-sm">
                    <span className="text-muted-foreground">Token:</span>
                    <div className="flex items-center gap-2">
                      <code className="text-xs bg-muted px-2 py-1 rounded">
                        {createdProject?.projectToken}
                      </code>
                      <Button
                        variant="ghost"
                        size="icon"
                        className="h-6 w-6"
                        onClick={handleCopyToken}
                      >
                        <IconCopy className="h-3 w-3" />
                      </Button>
                    </div>
                  </div>
                </div>
              </div>

              <div className="grid gap-2">
                <div className="flex items-center justify-between">
                  <Label>Tracking Script</Label>
                  <Button
                    variant="ghost"
                    size="sm"
                    onClick={handleCopyScript}
                  >
                    <IconCopy className="mr-2 h-4 w-4" />
                    Copy Script
                  </Button>
                </div>
                <div className="rounded-lg border bg-muted/50 p-3">
                  <code className="text-xs text-muted-foreground whitespace-pre-wrap">
                    {`<!-- Zori Analytics -->
<script async src="https://analytics.zori.app/script.js" data-website-id="${createdProject?.projectToken}"></script>`}
                  </code>
                </div>
                <p className="text-sm text-muted-foreground">
                  Add this script to the{' '}
                  <code className="bg-muted px-1 rounded">
                    &lt;head&gt;
                  </code>{' '}
                  section of your website
                </p>
              </div>

              <div className="rounded-lg bg-blue-50 dark:bg-blue-950/30 border border-blue-200 dark:border-blue-800 p-3">
                <div className="flex gap-2">
                  <IconCode className="h-4 w-4 text-blue-600 dark:text-blue-400 mt-0.5" />
                  <div className="text-sm">
                    <p className="font-medium text-blue-900 dark:text-blue-100 mb-1">
                      Installation Instructions
                    </p>
                    <ol className="text-blue-800 dark:text-blue-200 space-y-1 list-decimal list-inside">
                      <li>Copy the tracking script above</li>
                      <li>
                        Paste it before the closing{' '}
                        <code className="bg-blue-100 dark:bg-blue-900/50 px-1 rounded">
                          &lt;/head&gt;
                        </code>{' '}
                        tag
                      </li>
                      <li>Deploy your changes</li>
                      <li>Data will start appearing within minutes</li>
                    </ol>
                  </div>
                </div>
              </div>
            </div>
            <SheetFooter>
              <Button
                variant="outline"
                onClick={() => {
                  setCreateStep('form')
                  resetForm()
                }}
              >
                Create Another
              </Button>
              <Button onClick={() => handleOpenChange(false)}>Done</Button>
            </SheetFooter>
          </>
        )}
      </SheetContent>
    </Sheet>
  )
}
