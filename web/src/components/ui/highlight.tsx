import { useMemo } from 'react'

interface HighlightProps {
  text: string
  query: string
  className?: string
}

export function Highlight({ text, query, className = 'bg-yellow-500/30' }: HighlightProps) {
  const parts = useMemo(() => {
    if (!query.trim()) {
      return [{ text, highlight: false }]
    }

    const regex = new RegExp(`(${escapeRegExp(query)})`, 'gi')
    const parts: { text: string; highlight: boolean }[] = []
    let lastIndex = 0

    text.replace(regex, (match, _p1, offset) => {
      // Add non-matching text before this match
      if (offset > lastIndex) {
        parts.push({ text: text.slice(lastIndex, offset), highlight: false })
      }
      // Add the matching text
      parts.push({ text: match, highlight: true })
      lastIndex = offset + match.length
      return match
    })

    // Add any remaining non-matching text
    if (lastIndex < text.length) {
      parts.push({ text: text.slice(lastIndex), highlight: false })
    }

    return parts
  }, [text, query])

  return (
    <>
      {parts.map((part, i) =>
        part.highlight ? (
          <mark key={i} className={className}>
            {part.text}
          </mark>
        ) : (
          <span key={i}>{part.text}</span>
        )
      )}
    </>
  )
}

function escapeRegExp(string: string): string {
  return string.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')
}
