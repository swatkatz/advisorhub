interface BadgeProps {
  children: React.ReactNode
  color: string
  bgColor: string
}

export function Badge({ children, color, bgColor }: BadgeProps) {
  return (
    <span
      style={{
        fontSize: 11,
        fontWeight: 600,
        letterSpacing: '0.04em',
        textTransform: 'uppercase',
        color,
        background: bgColor,
        padding: '3px 8px',
        borderRadius: 4,
      }}
    >
      {children}
    </span>
  )
}
