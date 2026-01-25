import { useEffect, useRef, useCallback, type ReactNode } from 'react'
import { X } from 'lucide-react'
import { Card } from './card'
import { Button } from './button'
import { cn } from '@/lib/utils'

interface ModalProps {
  open: boolean
  onClose: () => void
  title?: string
  titleId?: string
  children: ReactNode
  className?: string
}

export function Modal({ open, onClose, title, titleId, children, className }: ModalProps) {
  const modalRef = useRef<HTMLDivElement>(null)
  const previousActiveElement = useRef<Element | null>(null)

  // Store the previously focused element and restore it on close
  useEffect(() => {
    if (open) {
      previousActiveElement.current = document.activeElement
    } else if (previousActiveElement.current instanceof HTMLElement) {
      previousActiveElement.current.focus()
    }
  }, [open])

  // Focus trap implementation
  const handleKeyDown = useCallback((e: KeyboardEvent) => {
    if (e.key === 'Escape') {
      e.preventDefault()
      onClose()
      return
    }

    if (e.key !== 'Tab' || !modalRef.current) return

    const focusableElements = modalRef.current.querySelectorAll<HTMLElement>(
      'button, [href], input, select, textarea, [tabindex]:not([tabindex="-1"])'
    )
    const firstElement = focusableElements[0]
    const lastElement = focusableElements[focusableElements.length - 1]

    // Trap focus within modal
    if (e.shiftKey) {
      if (document.activeElement === firstElement) {
        e.preventDefault()
        lastElement?.focus()
      }
    } else {
      if (document.activeElement === lastElement) {
        e.preventDefault()
        firstElement?.focus()
      }
    }
  }, [onClose])

  // Set up focus trap and body scroll lock
  useEffect(() => {
    if (!open) return

    // Lock body scroll
    const originalOverflow = document.body.style.overflow
    document.body.style.overflow = 'hidden'

    // Add keyboard listener
    document.addEventListener('keydown', handleKeyDown)

    // Focus the modal
    const timer = setTimeout(() => {
      const focusableElements = modalRef.current?.querySelectorAll<HTMLElement>(
        'button, [href], input, select, textarea, [tabindex]:not([tabindex="-1"])'
      )
      // Focus the close button or first focusable element
      focusableElements?.[0]?.focus()
    }, 0)

    return () => {
      document.body.style.overflow = originalOverflow
      document.removeEventListener('keydown', handleKeyDown)
      clearTimeout(timer)
    }
  }, [open, handleKeyDown])

  if (!open) return null

  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center bg-black/50"
      onClick={(e) => {
        if (e.target === e.currentTarget) onClose()
      }}
      role="dialog"
      aria-modal="true"
      aria-labelledby={titleId}
    >
      <Card
        ref={modalRef}
        className={cn('w-full max-w-2xl max-h-[80vh] overflow-auto', className)}
        onClick={(e) => e.stopPropagation()}
      >
        {title && (
          <div className="flex items-center justify-between p-6 pb-4">
            <h2 id={titleId} className="text-xl font-semibold">
              {title}
            </h2>
            <Button
              variant="ghost"
              size="icon"
              onClick={onClose}
              aria-label="Close dialog"
            >
              <X className="h-5 w-5" />
            </Button>
          </div>
        )}
        {children}
      </Card>
    </div>
  )
}
