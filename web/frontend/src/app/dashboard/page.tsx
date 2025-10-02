import data from './data.json'
import { ChartAreaInteractive } from '@/components/chart-area-interactive'
import { DataTable } from '@/components/data-table'
import { Navbar } from '@/components/navbar'

export default function DashboardPage() {
  return (
    <>
      <Navbar />
      <div className="flex flex-1 flex-col pl-[20%] pr-[20%]">
        <div className="@container/main flex flex-1 flex-col gap-2">
          <div className="flex flex-col gap-4 py-4 md:gap-6 md:py-6">
            <div className="px-4 lg:px-6">
              <ChartAreaInteractive />
            </div>
            <DataTable data={data} />
          </div>
        </div>
      </div>
    </>
  )
}
