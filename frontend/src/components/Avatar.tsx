interface AvatarProps {
  initials: string
  size?: number
}

export function Avatar({ initials, size = 36 }: AvatarProps) {
  const hue = initials.charCodeAt(0) * 7 + initials.charCodeAt(1) * 13
  const bg = `hsl(${hue % 360}, 40%, 25%)`
  const fg = `hsl(${hue % 360}, 60%, 75%)`

  return (
    <div
      style={{
        width: size,
        height: size,
        borderRadius: '50%',
        background: bg,
        color: fg,
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        fontSize: size * 0.36,
        fontWeight: 600,
        letterSpacing: '0.02em',
        flexShrink: 0,
      }}
    >
      {initials}
    </div>
  )
}
