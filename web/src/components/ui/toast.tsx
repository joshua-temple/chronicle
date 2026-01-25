import { useEffect, useRef } from 'react'
import { X, CheckCircle2, XCircle, AlertCircle, Info } from 'lucide-react'
import { cn } from '@/lib/utils'
import { useToastStore, type ToastType } from '@/stores/toast'

const TOAST_ICONS: Record<ToastType, React.ReactNode> = {
  success: <CheckCircle2 className="h-5 w-5 text-green-500" />,
  error: <XCircle className="h-5 w-5 text-red-500" />,
  warning: <AlertCircle className="h-5 w-5 text-yellow-500" />,
  info: <Info className="h-5 w-5 text-blue-500" />,
}

const TOAST_STYLES: Record<ToastType, string> = {
  success: 'border-green-500/50 bg-green-500/10',
  error: 'border-red-500/50 bg-red-500/10',
  warning: 'border-yellow-500/50 bg-yellow-500/10',
  info: 'border-blue-500/50 bg-blue-500/10',
}

export function ToastContainer() {
  const toasts = useToastStore((state) => state.toasts)
  const removeToast = useToastStore((state) => state.removeToast)

  if (toasts.length === 0) return null

  return (
    <div
      className="fixed bottom-4 right-4 z-50 flex flex-col gap-2"
      role="region"
      aria-label="Notifications"
      aria-live="polite"
    >
      {toasts.map((toast) => (
        <ToastItem key={toast.id} toast={toast} onDismiss={() => removeToast(toast.id)} />
      ))}
    </div>
  )
}

interface ToastItemProps {
  toast: {
    id: string
    type: ToastType
    title: string
    description?: string
  }
  onDismiss: () => void
}

function ToastItem({ toast, onDismiss }: ToastItemProps) {
  const ref = useRef<HTMLDivElement>(null)

  useEffect(() => {
    // Focus management for accessibility
    ref.current?.focus()
  }, [toast])

  return (
    <div
      ref={ref}
      role="alert"
      tabIndex={-1}
      className={cn(
        'flex items-start gap-3 rounded-lg border p-4 shadow-lg backdrop-blur-sm',
        'animate-in slide-in-from-right-full duration-300',
        'min-w-[300px] max-w-[400px]',
        TOAST_STYLES[toast.type]
      )}
    >
      <div className="flex-shrink-0">{TOAST_ICONS[toast.type]}</div>
      <div className="flex-1 min-w-0">
        <p className="font-medium text-foreground">{toast.title}</p>
        {toast.description && (
          <p className="mt-1 text-sm text-muted-foreground">{toast.description}</p>
        )}
      </div>
      <button
        onClick={onDismiss}
        className="flex-shrink-0 rounded p-1 hover:bg-white/10 transition-colors"
        aria-label="Dismiss notification"
      >
        <X className="h-4 w-4 text-muted-foreground" />
      </button>
    </div>
  )
}
