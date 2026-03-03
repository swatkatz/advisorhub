import { colors } from '../../styles/theme'
import { AccountTag } from '../../components/AccountTag'
import type { AccountContribution } from '../../generated/graphql'

interface ContributionBarProps {
  contribution: AccountContribution
}

export function ContributionBar({ contribution }: ContributionBarProps) {
  const { accountType, annualLimit, contributed, remaining, isOverContributed, overAmount, penaltyPerMonth } = contribution
  const pct = Math.min((contributed / annualLimit) * 100, 100)

  const barColor = isOverContributed ? colors.red : remaining === 0 ? colors.green : colors.accent
  const labelColor = isOverContributed ? colors.red : remaining === 0 ? colors.green : colors.textMuted

  let label: string
  if (isOverContributed && overAmount) {
    label = `Over by $${Math.abs(overAmount).toLocaleString()}`
  } else if (remaining === 0) {
    label = 'Maxed ✓'
  } else {
    label = `$${remaining.toLocaleString()} room`
  }

  return (
    <div style={{ marginBottom: 10 }}>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 4 }}>
        <AccountTag type={accountType} />
        <span style={{ fontSize: 11, color: labelColor, fontWeight: 600 }}>{label}</span>
      </div>
      <div style={{ height: 6, background: colors.surfaceRaised, borderRadius: 3, overflow: 'hidden', position: 'relative' }}>
        <div
          style={{
            position: 'absolute',
            left: 0,
            top: 0,
            height: '100%',
            borderRadius: 3,
            transition: 'width 0.5s ease',
            width: `${Math.min(pct, 100)}%`,
            background: barColor,
          }}
        />
      </div>
      <div style={{ display: 'flex', justifyContent: 'space-between', marginTop: 3 }}>
        <span style={{ fontSize: 10, color: colors.textDim }}>${contributed.toLocaleString()} contributed</span>
        <span style={{ fontSize: 10, color: colors.textDim }}>${annualLimit.toLocaleString()} limit</span>
      </div>
      {isOverContributed && penaltyPerMonth != null && penaltyPerMonth > 0 && (
        <div style={{ fontSize: 10, color: colors.red, marginTop: 2 }}>
          Penalty: ${penaltyPerMonth.toLocaleString()}/month
        </div>
      )}
    </div>
  )
}
