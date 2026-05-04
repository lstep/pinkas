import React from 'react'

export interface BadgeProps {
  children: React.ReactNode
  variant?: 'default' | 'admin' | 'editor' | 'viewer'
  size?: 'sm' | 'md'
}

export const Badge: React.FC<BadgeProps> = ({ 
  children, 
  variant = 'default',
  size = 'md'
}) => {
  const variantClass = variant !== 'default' ? `badge-${variant}` : ''
  const sizeClass = size !== 'md' ? `badge-${size}` : ''
  
  const combinedClassName = `badge ${variantClass} ${sizeClass}`.trim()
  
  return (
    <span className={combinedClassName}>
      {children}
    </span>
  )
}
