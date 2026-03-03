import { severityColors } from '../styles/theme'

interface SeverityDotProps {
  severity: string
}

export function SeverityDot({ severity }: SeverityDotProps) {
  const c = severityColors[severity] || severityColors['INFO']
  return (
    <span
      style={{
        width: 8,
        height: 8,
        borderRadius: '50%',
        background: c.dot,
        boxShadow: `0 0 6px ${c.dot}`,
        flexShrink: 0,
        marginTop: 6,
        display: 'inline-block',
      }}
    />
  )
}
