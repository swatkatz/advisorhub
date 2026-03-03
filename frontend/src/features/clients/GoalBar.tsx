import { colors } from '../../styles/theme'
import type { GetClientQuery } from '../../generated/graphql'

type GoalData = GetClientQuery['client']['goals'][number]

interface GoalBarProps {
  goal: GoalData
}

export function GoalBar({ goal }: GoalBarProps) {
  const goalColor = goal.status === 'BEHIND' ? colors.yellow : colors.green

  return (
    <div style={{ marginBottom: 10 }}>
      <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: 4 }}>
        <span style={{ fontSize: 12, color: colors.text }}>{goal.name}</span>
        <span style={{ fontSize: 11, fontWeight: 600, color: goalColor }}>{goal.progressPct}%</span>
      </div>
      <div style={{ height: 4, background: colors.surfaceRaised, borderRadius: 2 }}>
        <div
          style={{
            height: '100%',
            width: `${goal.progressPct}%`,
            background: goalColor,
            borderRadius: 2,
            transition: 'width 0.5s ease',
          }}
        />
      </div>
    </div>
  )
}
