import { useState } from 'react'
import {
  IconChartLine,
  IconCheck,
  IconChevronLeft,
  IconChevronRight,
} from '@tabler/icons-react'
import { toast } from 'sonner'
import { ProjectDetailsStep } from './project-details-step'
import { ProjectSetupStep } from './project-setup-step'
import { ProjectCompleteStep } from './project-complete-step'
import { Card } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { useCreateProject } from '@/hooks/use-projects'

export interface ProjectFormData {
  name: string
  websiteUrl: string
  allowLocalhost: boolean
}

export interface CreatedProject {
  id: string
  projectToken: string
  name: string
  domain: string
}

type OnboardingStep = 'details' | 'setup' | 'complete'

const steps = [
  {
    id: 'details',
    title: 'Project Details',
    description: 'Enter your project information',
  },
  {
    id: 'setup',
    title: 'Setup Tracking',
    description: 'Configure and install the tracking script',
  },
  {
    id: 'complete',
    title: 'Complete',
    description: 'Start viewing your analytics',
  },
] as const

export function ProjectOnboarding() {
  const [currentStep, setCurrentStep] = useState<OnboardingStep>('details')
  const [projectData, setProjectData] = useState<ProjectFormData>({
    name: '',
    websiteUrl: '',
    allowLocalhost: false,
  })
  const [createdProject, setCreatedProject] = useState<CreatedProject | null>(
    null,
  )

  const createProjectMutation = useCreateProject()

  const currentStepIndex = steps.findIndex((step) => step.id === currentStep)
  const isLastStep = currentStepIndex === steps.length - 1
  const isFirstStep = currentStepIndex === 0

  const handleNext = async () => {
    if (currentStep === 'details') {
      // Validate form data
      if (!projectData.name || !projectData.websiteUrl) {
        toast.error('Please fill in all required fields')
        return
      }

      // Create the project
      try {
        const result = await createProjectMutation.mutateAsync({
          name: projectData.name,
          website_url: projectData.websiteUrl,
          allow_localhost: projectData.allowLocalhost,
        })

        setCreatedProject({
          id: result.id,
          projectToken: result.project_token,
          name: result.name,
          domain: result.domain,
        })

        setCurrentStep('setup')
        toast.success('Project created successfully!')
      } catch (error: any) {
        toast.error('Failed to create project: ' + error.message)
      }
    } else if (currentStep === 'setup') {
      setCurrentStep('complete')
    }
  }

  const handleBack = () => {
    if (currentStep === 'setup') {
      setCurrentStep('details')
    } else if (currentStep === 'complete') {
      setCurrentStep('setup')
    }
  }

  const renderStepContent = () => {
    switch (currentStep) {
      case 'details':
        return (
          <ProjectDetailsStep data={projectData} onChange={setProjectData} />
        )
      case 'setup':
        return <ProjectSetupStep project={createdProject} />
      case 'complete':
        return <ProjectCompleteStep project={createdProject} />
      default:
        return null
    }
  }

  return (
    <div className="min-h-screen">
      <div className="container mx-auto px-4 py-8">
        {/* Header */}

        {/* Progress Steps */}
        <div className="max-w-3xl mx-auto mb-8">
          <div className="flex items-center justify-between">
            {steps.map((step, index) => (
              <div key={step.id} className="flex items-center">
                <div className="flex flex-col items-center">
                  <div
                    className={`
                    w-10 h-10 rounded-full flex items-center justify-center border-2 transition-all duration-300 font-mono
                    ${
                      index < currentStepIndex
                        ? 'bg-accent border-accent text-accent-foreground shadow-sm'
                        : index === currentStepIndex
                          ? 'bg-primary border-primary text-primary-foreground shadow-md'
                          : 'bg-muted/30 border-border text-muted-foreground'
                    }
                  `}
                  >
                    {index < currentStepIndex ? (
                      <IconCheck className="w-5 h-5" />
                    ) : (
                      <span className="text-sm font-medium">{index + 1}</span>
                    )}
                  </div>
                  <div className="text-center mt-3">
                    <p
                      className={`text-sm font-medium font-mono tracking-tight ${
                        index <= currentStepIndex
                          ? 'text-foreground'
                          : 'text-muted-foreground'
                      }`}
                    >
                      {step.title}
                    </p>
                    <p className="text-xs text-muted-foreground hidden sm:block mt-1 font-mono">
                      {step.description}
                    </p>
                  </div>
                </div>
                {index < steps.length - 1 && (
                  <div
                    className={`
                    h-0.5 w-16 md:w-24 mx-4 transition-all duration-300 rounded-full
                    ${index < currentStepIndex ? 'bg-accent' : 'bg-border'}
                  `}
                  />
                )}
              </div>
            ))}
          </div>
        </div>

        {/* Step Content */}
        <div className="max-w-5xl mx-auto">
          <Card className="p-8 shadow-lg border-border/50">
            {renderStepContent()}
          </Card>

          {/* Navigation */}
          <div className="flex justify-between items-center mt-6">
            <Button
              variant="outline"
              onClick={handleBack}
              disabled={isFirstStep}
              className="flex items-center gap-2 font-mono tracking-tight shadow-sm"
            >
              <IconChevronLeft className="w-4 h-4" />
              Previous
            </Button>

            <div className="text-sm text-muted-foreground font-mono">
              Step {currentStepIndex + 1} of {steps.length}
            </div>

            {!isLastStep ? (
              <Button
                onClick={handleNext}
                disabled={createProjectMutation.isPending}
                className="flex items-center gap-2 font-mono tracking-tight shadow-sm"
              >
                {createProjectMutation.isPending ? 'Creating...' : 'Next'}
                <IconChevronRight className="w-4 h-4" />
              </Button>
            ) : (
              <Button
                onClick={() => window.location.reload()}
                className="flex items-center gap-2 font-mono tracking-tight shadow-sm"
              >
                View Dashboard
                <IconChevronRight className="w-4 h-4" />
              </Button>
            )}
          </div>
        </div>
      </div>
    </div>
  )
}
