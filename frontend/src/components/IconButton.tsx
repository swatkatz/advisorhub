import { useState } from 'react'
import { colors } from '../styles/theme'

interface IconButtonProps {
  children: React.ReactNode
  onClick?: () => void
  title?: string
  variant?: 'default' | 'send' | 'dismiss'
  disabled?: boolean
}

const variantStyles = {
  default: {
    bg: 'transparent',
    bgHover: colors.surfaceRaised,
    border: colors.borderLight,
    color: colors.textMuted,
  },
  send: {
    bg: colors.accentDim,
    bgHover: 'rgba(34,211,238,0.15)',
    border: 'rgba(34,211,238,0.3)',
    color: colors.accent,
  },
  dismiss: {
    bg: 'transparent',
    bgHover: 'rgba(239,68,68,0.12)',
    border: colors.borderLight,
    color: colors.textMuted,
  },
}

export function IconButton({
  children,
  onClick,
  title,
  variant = 'default',
  disabled = false,
}: IconButtonProps) {
  const [hovered, setHovered] = useState(false)
  const s = variantStyles[variant]

  return (
    <button
      onClick={onClick}
      title={title}
      disabled={disabled}
      onMouseEnter={() => setHovered(true)}
      onMouseLeave={() => setHovered(false)}
      style={{
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        width: 32,
        height: 32,
        borderRadius: 6,
        background: hovered ? s.bgHover : s.bg,
        border: `1px solid ${s.border}`,
        color: s.color,
        cursor: disabled ? 'default' : 'pointer',
        transition: 'all 0.15s',
        opacity: disabled ? 0.5 : 1,
      }}
    >
      {children}
    </button>
  )
}
