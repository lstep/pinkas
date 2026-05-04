import React from 'react'

export interface InputProps extends React.InputHTMLAttributes<HTMLInputElement> {
  label?: string
  error?: string
  hint?: string
}

export const Input = React.forwardRef<HTMLInputElement, InputProps>(
  ({ label, error, hint, className = '', ...props }, ref) => {
    const hasError = !!error
    
    return (
      <div className="input-group">
        {label && (
          <label htmlFor={props.id}>
            {label}
            {props.required && <span className="input-required"> *</span>}
          </label>
        )}
        <input
          ref={ref}
          className={className}
          aria-invalid={hasError}
          aria-describedby={error ? `${props.id}-error` : hint ? `${props.id}-hint` : undefined}
          {...props}
        />
        {error && (
          <span id={`${props.id}-error`} className="input-error" role="alert">
            {error}
          </span>
        )}
        {hint && !error && (
          <span id={`${props.id}-hint`} className="input-hint">
            {hint}
          </span>
        )}
      </div>
    )
  }
)

Input.displayName = 'Input'
