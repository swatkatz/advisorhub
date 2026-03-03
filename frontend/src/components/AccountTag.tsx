import { accountTypeColors } from '../styles/theme'

interface AccountTagProps {
  type: string
}

const displayLabels: Record<string, string> = {
  RRSP: 'RRSP',
  TFSA: 'TFSA',
  FHSA: 'FHSA',
  RESP: 'RESP',
  NON_REG: 'Non-Reg',
}

export function AccountTag({ type }: AccountTagProps) {
  const c = accountTypeColors[type] || accountTypeColors['NON_REG']
  return (
    <span
      style={{
        fontSize: 10,
        fontWeight: 600,
        padding: '2px 6px',
        borderRadius: 3,
        background: c.bg,
        color: c.text,
        letterSpacing: '0.03em',
      }}
    >
      {displayLabels[type] || type}
    </span>
  )
}
