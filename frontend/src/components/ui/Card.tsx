import React from 'react'

export interface CardProps {
  children: React.ReactNode
  className?: string
  padding?: 'none' | 'sm' | 'md' | 'lg'
  hover?: boolean
}

export const Card: React.FC<CardProps> = ({ 
  children, 
  className = '', 
  padding = 'md',
  hover = false 
}) => {
  const paddingClass = padding !== 'md' ? `card-padding-${padding}` : ''
  const hoverClass = hover ? 'card-hover' : ''
  
  const combinedClassName = `card ${paddingClass} ${hoverClass} ${className}`.trim()
  
  return (
    <div className={combinedClassName}>
      {children}
    </div>
  )
}
