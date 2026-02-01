import { describe, it, expect } from 'vitest'
import { render, screen } from '@testing-library/react'
import { Highlight } from './highlight'

describe('Highlight Component', () => {
  describe('Rendering', () => {
    it('renders text without highlight when query is empty', () => {
      render(<Highlight text="Hello World" query="" />)
      expect(screen.getByText('Hello World')).toBeInTheDocument()
    })

    it('renders text without highlight when query is whitespace only', () => {
      render(<Highlight text="Hello World" query="   " />)
      expect(screen.getByText('Hello World')).toBeInTheDocument()
    })

    it('highlights matching text', () => {
      const { container } = render(<Highlight text="Hello World" query="World" />)
      const mark = container.querySelector('mark')
      expect(mark).toBeInTheDocument()
      expect(mark).toHaveTextContent('World')
    })

    it('preserves non-matching text', () => {
      const { container } = render(<Highlight text="Hello World" query="World" />)
      // Check that both the non-matching and matching text are present
      expect(container.textContent).toBe('Hello World')
      // Span contains the text before "World", mark contains "World"
      expect(container.querySelector('span')?.textContent).toContain('Hello')
      expect(container.querySelector('mark')?.textContent).toBe('World')
    })
  })

  describe('Case insensitivity', () => {
    it('highlights regardless of case', () => {
      const { container } = render(<Highlight text="Hello World" query="world" />)
      const mark = container.querySelector('mark')
      expect(mark).toBeInTheDocument()
      expect(mark).toHaveTextContent('World')
    })

    it('highlights uppercase query in lowercase text', () => {
      const { container } = render(<Highlight text="hello world" query="WORLD" />)
      const mark = container.querySelector('mark')
      expect(mark).toBeInTheDocument()
      expect(mark).toHaveTextContent('world')
    })

    it('handles mixed case', () => {
      const { container } = render(<Highlight text="HeLLo WoRLd" query="hello" />)
      const mark = container.querySelector('mark')
      expect(mark).toBeInTheDocument()
      expect(mark).toHaveTextContent('HeLLo')
    })
  })

  describe('Multiple matches', () => {
    it('highlights all occurrences', () => {
      const { container } = render(<Highlight text="foo bar foo baz foo" query="foo" />)
      const marks = container.querySelectorAll('mark')
      expect(marks.length).toBe(3)
    })

    it('preserves text between matches', () => {
      const { container } = render(<Highlight text="foo bar foo" query="foo" />)
      // Check that the text between matches is preserved
      expect(container.textContent).toBe('foo bar foo')
      const span = container.querySelector('span')
      // The span contains " bar " but toHaveTextContent may normalize whitespace
      expect(span?.textContent).toContain('bar')
    })
  })

  describe('Custom className', () => {
    it('applies default highlight class', () => {
      const { container } = render(<Highlight text="Hello World" query="World" />)
      const mark = container.querySelector('mark')
      expect(mark).toHaveClass('bg-yellow-500/30')
    })

    it('applies custom highlight class', () => {
      const { container } = render(
        <Highlight text="Hello World" query="World" className="bg-blue-500" />
      )
      const mark = container.querySelector('mark')
      expect(mark).toHaveClass('bg-blue-500')
      expect(mark).not.toHaveClass('bg-yellow-500/30')
    })
  })

  describe('Special characters', () => {
    it('handles regex special characters in query', () => {
      const { container } = render(<Highlight text="test (value)" query="(value)" />)
      const mark = container.querySelector('mark')
      expect(mark).toBeInTheDocument()
      expect(mark).toHaveTextContent('(value)')
    })

    it('handles dot in query', () => {
      const { container } = render(<Highlight text="file.txt" query="." />)
      const mark = container.querySelector('mark')
      expect(mark).toBeInTheDocument()
      expect(mark).toHaveTextContent('.')
    })

    it('handles asterisk in query', () => {
      const { container } = render(<Highlight text="a*b" query="*" />)
      const mark = container.querySelector('mark')
      expect(mark).toBeInTheDocument()
      expect(mark).toHaveTextContent('*')
    })

    it('handles plus in query', () => {
      const { container } = render(<Highlight text="a+b" query="+" />)
      const mark = container.querySelector('mark')
      expect(mark).toBeInTheDocument()
      expect(mark).toHaveTextContent('+')
    })

    it('handles question mark in query', () => {
      const { container } = render(<Highlight text="really?" query="?" />)
      const mark = container.querySelector('mark')
      expect(mark).toBeInTheDocument()
      expect(mark).toHaveTextContent('?')
    })

    it('handles brackets in query', () => {
      const { container } = render(<Highlight text="array[0]" query="[0]" />)
      const mark = container.querySelector('mark')
      expect(mark).toBeInTheDocument()
      expect(mark).toHaveTextContent('[0]')
    })

    it('handles backslash in query', () => {
      const { container } = render(<Highlight text="path\\to\\file" query="\\" />)
      const marks = container.querySelectorAll('mark')
      expect(marks.length).toBe(2)
    })

    it('handles pipe in query', () => {
      const { container } = render(<Highlight text="a|b" query="|" />)
      const mark = container.querySelector('mark')
      expect(mark).toBeInTheDocument()
      expect(mark).toHaveTextContent('|')
    })

    it('handles caret in query', () => {
      const { container } = render(<Highlight text="^start" query="^" />)
      const mark = container.querySelector('mark')
      expect(mark).toBeInTheDocument()
      expect(mark).toHaveTextContent('^')
    })

    it('handles dollar sign in query', () => {
      const { container } = render(<Highlight text="$money" query="$" />)
      const mark = container.querySelector('mark')
      expect(mark).toBeInTheDocument()
      expect(mark).toHaveTextContent('$')
    })

    it('handles curly braces in query', () => {
      const { container } = render(<Highlight text="{code}" query="{" />)
      const mark = container.querySelector('mark')
      expect(mark).toBeInTheDocument()
      expect(mark).toHaveTextContent('{')
    })
  })

  describe('Edge cases', () => {
    it('handles empty text', () => {
      const { container } = render(<Highlight text="" query="test" />)
      const mark = container.querySelector('mark')
      expect(mark).not.toBeInTheDocument()
    })

    it('handles no match', () => {
      const { container } = render(<Highlight text="Hello World" query="xyz" />)
      const mark = container.querySelector('mark')
      expect(mark).not.toBeInTheDocument()
      expect(screen.getByText('Hello World')).toBeInTheDocument()
    })

    it('handles text equal to query', () => {
      const { container } = render(<Highlight text="test" query="test" />)
      const mark = container.querySelector('mark')
      expect(mark).toBeInTheDocument()
      expect(mark).toHaveTextContent('test')
    })

    it('handles query longer than text', () => {
      const { container } = render(<Highlight text="hi" query="hello world" />)
      const mark = container.querySelector('mark')
      expect(mark).not.toBeInTheDocument()
    })

    it('handles single character query', () => {
      const { container } = render(<Highlight text="abcabc" query="a" />)
      const marks = container.querySelectorAll('mark')
      expect(marks.length).toBe(2)
    })

    it('handles query at start of text', () => {
      const { container } = render(<Highlight text="test string" query="test" />)
      const mark = container.querySelector('mark')
      expect(mark).toBeInTheDocument()
      expect(mark).toHaveTextContent('test')
    })

    it('handles query at end of text', () => {
      const { container } = render(<Highlight text="hello test" query="test" />)
      const mark = container.querySelector('mark')
      expect(mark).toBeInTheDocument()
      expect(mark).toHaveTextContent('test')
    })

    it('handles overlapping potential matches', () => {
      const { container } = render(<Highlight text="aaa" query="aa" />)
      // Should only match once (first occurrence)
      const marks = container.querySelectorAll('mark')
      expect(marks.length).toBe(1)
      expect(marks[0]).toHaveTextContent('aa')
    })
  })

  describe('Memoization', () => {
    it('computes parts correctly on initial render', () => {
      const { container } = render(<Highlight text="foo bar foo" query="foo" />)
      const marks = container.querySelectorAll('mark')
      const spans = container.querySelectorAll('span')

      expect(marks.length).toBe(2) // Two "foo" matches
      expect(spans.length).toBe(1) // One " bar " span
    })
  })

  describe('Accessibility', () => {
    it('uses mark element for highlights', () => {
      const { container } = render(<Highlight text="Hello World" query="World" />)
      const mark = container.querySelector('mark')
      expect(mark).toBeInTheDocument()
    })

    it('preserves text content', () => {
      const { container } = render(<Highlight text="Hello World" query="World" />)
      expect(container.textContent).toBe('Hello World')
    })
  })
})
