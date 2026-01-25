import { useState } from 'react'
import { useResults, useResult, useDeleteResult } from '@/hooks/useResults'
import { ResultDetail } from '@/components/results/ResultDetail'
import { Card, CardContent } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Loader2, Trash2, ChevronRight, CheckCircle2, XCircle } from 'lucide-react'

export function Results() {
  const { data, isLoading } = useResults()
  const deleteResult = useDeleteResult()
  const [selectedResult, setSelectedResult] = useState<string | null>(null)

  if (isLoading) {
    return (
      <div className="flex items-center justify-center p-8">
        <Loader2 className="h-8 w-8 animate-spin" />
      </div>
    )
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h1 className="text-3xl font-bold">Results</h1>
        <span className="text-muted-foreground">{data?.count || 0} total</span>
      </div>

      {data?.results?.length === 0 ? (
        <Card>
          <CardContent className="p-8 text-center text-muted-foreground">
            No results yet. Run a scenario to see results here.
          </CardContent>
        </Card>
      ) : (
        <div className="space-y-2">
          {data?.results?.map((id) => (
            <ResultRow
              key={id}
              id={id}
              onSelect={setSelectedResult}
              onDelete={(id) => deleteResult.mutate(id)}
            />
          ))}
        </div>
      )}

      {selectedResult && (
        <ResultDetail id={selectedResult} onClose={() => setSelectedResult(null)} />
      )}
    </div>
  )
}

function ResultRow({
  id,
  onSelect,
  onDelete,
}: {
  id: string
  onSelect: (id: string) => void
  onDelete: (id: string) => void
}) {
  const { data: result } = useResult(id)

  return (
    <div
      className="flex items-center justify-between rounded-lg border border-border p-4 cursor-pointer hover:bg-secondary/50 transition-colors"
      onClick={() => onSelect(id)}
      role="button"
      tabIndex={0}
      onKeyDown={(e) => e.key === 'Enter' && onSelect(id)}
    >
      <div className="flex items-center gap-4">
        {result ? (
          result.failed > 0 ? (
            <XCircle className="h-5 w-5 text-red-500" />
          ) : (
            <CheckCircle2 className="h-5 w-5 text-green-500" />
          )
        ) : (
          <Loader2 className="h-5 w-5 animate-spin" />
        )}
        <div>
          <div className="font-medium">{id}</div>
          {result && (
            <div className="text-sm text-muted-foreground">
              {result.passed}/{result.total_scenarios} passed • {result.duration}
            </div>
          )}
        </div>
      </div>
      <div className="flex items-center gap-2">
        <Button
          variant="ghost"
          size="icon"
          aria-label="Delete result"
          onClick={(e) => {
            e.stopPropagation()
            onDelete(id)
          }}
        >
          <Trash2 className="h-4 w-4" />
        </Button>
        <ChevronRight className="h-5 w-5 text-muted-foreground" />
      </div>
    </div>
  )
}
