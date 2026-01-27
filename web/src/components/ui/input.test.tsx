import { describe, it, expect, vi } from 'vitest'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { createRef } from 'react'
import { Input } from './input'

describe('Input Component', () => {
  describe('Rendering', () => {
    it('renders an input element', () => {
      render(<Input aria-label="test input" />)
      expect(screen.getByRole('textbox')).toBeInTheDocument()
    })

    it('applies default classes', () => {
      render(<Input data-testid="input" />)
      const input = screen.getByTestId('input')
      expect(input).toHaveClass('flex', 'h-10', 'w-full', 'rounded-md', 'border')
    })

    it('applies custom className', () => {
      render(<Input className="custom-class" data-testid="input" />)
      expect(screen.getByTestId('input')).toHaveClass('custom-class')
    })

    it('forwards ref', () => {
      const ref = createRef<HTMLInputElement>()
      render(<Input ref={ref} />)
      expect(ref.current).toBeInstanceOf(HTMLInputElement)
    })

    it('spreads additional props', () => {
      render(<Input data-testid="input" placeholder="Enter text" />)
      expect(screen.getByTestId('input')).toHaveAttribute('placeholder', 'Enter text')
    })

    it('has displayName', () => {
      expect(Input.displayName).toBe('Input')
    })
  })

  describe('Types', () => {
    it('renders text type by default', () => {
      render(<Input data-testid="input" />)
      const input = screen.getByTestId('input')
      // Text inputs have type="text" or no type attribute
      expect(input).not.toHaveAttribute('type', 'password')
    })

    it('renders password type', () => {
      render(<Input type="password" data-testid="input" />)
      expect(screen.getByTestId('input')).toHaveAttribute('type', 'password')
    })

    it('renders email type', () => {
      render(<Input type="email" data-testid="input" />)
      expect(screen.getByTestId('input')).toHaveAttribute('type', 'email')
    })

    it('renders number type', () => {
      render(<Input type="number" data-testid="input" />)
      expect(screen.getByTestId('input')).toHaveAttribute('type', 'number')
    })

    it('renders search type', () => {
      render(<Input type="search" data-testid="input" />)
      expect(screen.getByTestId('input')).toHaveAttribute('type', 'search')
    })

    it('renders tel type', () => {
      render(<Input type="tel" data-testid="input" />)
      expect(screen.getByTestId('input')).toHaveAttribute('type', 'tel')
    })

    it('renders url type', () => {
      render(<Input type="url" data-testid="input" />)
      expect(screen.getByTestId('input')).toHaveAttribute('type', 'url')
    })
  })

  describe('States', () => {
    it('can be disabled', () => {
      render(<Input disabled data-testid="input" />)
      expect(screen.getByTestId('input')).toBeDisabled()
    })

    it('applies disabled styles', () => {
      render(<Input disabled data-testid="input" />)
      expect(screen.getByTestId('input')).toHaveClass('disabled:cursor-not-allowed', 'disabled:opacity-50')
    })

    it('can be readonly', () => {
      render(<Input readOnly data-testid="input" />)
      expect(screen.getByTestId('input')).toHaveAttribute('readonly')
    })

    it('can be required', () => {
      render(<Input required data-testid="input" />)
      expect(screen.getByTestId('input')).toBeRequired()
    })

    it('applies focus-visible ring styles', () => {
      render(<Input data-testid="input" />)
      expect(screen.getByTestId('input')).toHaveClass('focus-visible:ring-2')
    })
  })

  describe('Value handling', () => {
    it('accepts defaultValue', () => {
      render(<Input defaultValue="default text" data-testid="input" />)
      expect(screen.getByTestId('input')).toHaveValue('default text')
    })

    it('accepts controlled value', () => {
      render(<Input value="controlled value" onChange={() => {}} data-testid="input" />)
      expect(screen.getByTestId('input')).toHaveValue('controlled value')
    })

    it('updates value on user input', async () => {
      const user = userEvent.setup()
      render(<Input data-testid="input" aria-label="input" />)

      const input = screen.getByTestId('input')
      await user.type(input, 'hello world')

      expect(input).toHaveValue('hello world')
    })
  })

  describe('Interactions', () => {
    it('calls onChange when value changes', async () => {
      const user = userEvent.setup()
      const handleChange = vi.fn()
      render(<Input onChange={handleChange} data-testid="input" />)

      await user.type(screen.getByTestId('input'), 'a')

      expect(handleChange).toHaveBeenCalled()
    })

    it('calls onFocus when focused', async () => {
      const user = userEvent.setup()
      const handleFocus = vi.fn()
      render(<Input onFocus={handleFocus} data-testid="input" />)

      await user.click(screen.getByTestId('input'))

      expect(handleFocus).toHaveBeenCalled()
    })

    it('calls onBlur when blurred', async () => {
      const user = userEvent.setup()
      const handleBlur = vi.fn()
      render(<Input onBlur={handleBlur} data-testid="input" />)

      const input = screen.getByTestId('input')
      await user.click(input)
      await user.tab()

      expect(handleBlur).toHaveBeenCalled()
    })

    it('can be focused with Tab', async () => {
      const user = userEvent.setup()
      render(<Input data-testid="input" />)

      await user.tab()

      expect(screen.getByTestId('input')).toHaveFocus()
    })

    it('does not fire onChange when disabled', async () => {
      const user = userEvent.setup()
      const handleChange = vi.fn()
      render(<Input disabled onChange={handleChange} data-testid="input" />)

      await user.type(screen.getByTestId('input'), 'test')

      expect(handleChange).not.toHaveBeenCalled()
    })
  })

  describe('Accessibility', () => {
    it('supports aria-label', () => {
      render(<Input aria-label="Custom label" />)
      expect(screen.getByLabelText('Custom label')).toBeInTheDocument()
    })

    it('supports aria-describedby', () => {
      render(
        <>
          <Input aria-describedby="description" data-testid="input" />
          <span id="description">Helper text</span>
        </>
      )
      expect(screen.getByTestId('input')).toHaveAttribute('aria-describedby', 'description')
    })

    it('supports aria-invalid', () => {
      render(<Input aria-invalid="true" data-testid="input" />)
      expect(screen.getByTestId('input')).toHaveAttribute('aria-invalid', 'true')
    })

    it('supports name attribute', () => {
      render(<Input name="email" data-testid="input" />)
      expect(screen.getByTestId('input')).toHaveAttribute('name', 'email')
    })

    it('supports id attribute', () => {
      render(<Input id="my-input" data-testid="input" />)
      expect(screen.getByTestId('input')).toHaveAttribute('id', 'my-input')
    })

    it('supports autocomplete attribute', () => {
      render(<Input autoComplete="email" data-testid="input" />)
      expect(screen.getByTestId('input')).toHaveAttribute('autocomplete', 'email')
    })
  })

  describe('Placeholder', () => {
    it('displays placeholder text', () => {
      render(<Input placeholder="Enter your name" />)
      expect(screen.getByPlaceholderText('Enter your name')).toBeInTheDocument()
    })

    it('applies placeholder styling class', () => {
      render(<Input placeholder="Test" data-testid="input" />)
      expect(screen.getByTestId('input')).toHaveClass('placeholder:text-muted-foreground')
    })
  })

  describe('File input', () => {
    it('renders file type', () => {
      render(<Input type="file" data-testid="input" />)
      expect(screen.getByTestId('input')).toHaveAttribute('type', 'file')
    })

    it('applies file input styles', () => {
      render(<Input type="file" data-testid="input" />)
      expect(screen.getByTestId('input')).toHaveClass('file:border-0', 'file:bg-transparent')
    })
  })
})
