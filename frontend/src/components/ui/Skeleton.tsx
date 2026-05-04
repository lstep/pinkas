import React from 'react'

export interface SkeletonProps {
  variant?: 'text' | 'circular' | 'rectangular'
  width?: string | number
  height?: string | number
  count?: number
}

export const Skeleton: React.FC<SkeletonProps> = ({ 
  variant = 'text',
  width,
  height,
  count = 1
}) => {
  const style: React.CSSProperties = {
    width: typeof width === 'number' ? `${width}px` : width,
    height: typeof height === 'number' ? `${height}px` : height,
  }
  
  // For text variant without explicit height, use default
  if (variant === 'text' && !height) {
    style.height = '1em'
  }
  
  // For circular, enforce equal width/height if only one is provided
  if (variant === 'circular') {
    if (width && !height) {
      style.height = style.width
    } else if (height && !width) {
      style.width = style.height
    }
    style.borderRadius = '50%'
  }
  
  const renderSkeleton = (index: number) => {
    // For text with count > 1, make last line 60% width
    const lineStyle = { ...style }
    if (variant === 'text' && count > 1 && index === count - 1) {
      lineStyle.width = '60%'
    }
    
    return (
      <div
        key={index}
        className={`skeleton skeleton-${variant}`}
        style={lineStyle}
      />
    )
  }
  
  if (count === 1) {
    return renderSkeleton(0)
  }
  
  return (
    <div className="skeleton-stack" style={{ display: 'flex', flexDirection: 'column', gap: '0.5em' }}>
      {Array.from({ length: count }, (_, i) => renderSkeleton(i))}
    </div>
  )
}
