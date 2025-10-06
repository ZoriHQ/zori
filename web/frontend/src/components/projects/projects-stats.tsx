import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'

interface ProjectsStatsProps {
  projects: Array<{
    status?: string
    visitors?: number
    pageViews?: number
  }>
}

export function ProjectsStats({ projects }: ProjectsStatsProps) {
  const activeProjects = projects.filter(
    (p) => p.status === 'active' || !p.status,
  ).length
  const totalVisitors = projects.reduce((sum, p) => sum + (p.visitors || 0), 0)
  const totalPageViews = projects.reduce(
    (sum, p) => sum + (p.pageViews || 0),
    0,
  )

  return (
    <div className="grid grid-cols-1 md:grid-cols-4 gap-4 mb-6">
      <Card>
        <CardHeader className="pb-2">
          <CardTitle className="text-sm font-medium text-muted-foreground">
            Total Projects
          </CardTitle>
        </CardHeader>
        <CardContent>
          <div className="text-2xl font-bold">{projects.length}</div>
        </CardContent>
      </Card>
      <Card>
        <CardHeader className="pb-2">
          <CardTitle className="text-sm font-medium text-muted-foreground">
            Active Projects
          </CardTitle>
        </CardHeader>
        <CardContent>
          <div className="text-2xl font-bold text-green-600 dark:text-green-400">
            {activeProjects}
          </div>
        </CardContent>
      </Card>
      <Card>
        <CardHeader className="pb-2">
          <CardTitle className="text-sm font-medium text-muted-foreground">
            Total Visitors
          </CardTitle>
        </CardHeader>
        <CardContent>
          <div className="text-2xl font-bold">
            {totalVisitors.toLocaleString()}
          </div>
        </CardContent>
      </Card>
      <Card>
        <CardHeader className="pb-2">
          <CardTitle className="text-sm font-medium text-muted-foreground">
            Total Page Views
          </CardTitle>
        </CardHeader>
        <CardContent>
          <div className="text-2xl font-bold">
            {totalPageViews.toLocaleString()}
          </div>
        </CardContent>
      </Card>
    </div>
  )
}
