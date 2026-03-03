import { colors } from '../../styles/theme'
import { AccountTag } from '../../components/AccountTag'
import type { GetTransfersQuery } from '../../generated/graphql'

type TransferData = GetTransfersQuery['transfers'][number]

interface TransferCardProps {
  transfer: TransferData
}

export function TransferCard({ transfer }: TransferCardProps) {
  return (
    <div
      style={{
        background: colors.surface,
        border: `1px solid ${transfer.isStuck ? 'rgba(239,68,68,0.4)' : colors.border}`,
        borderRadius: 8,
        padding: '10px 12px',
        marginBottom: 6,
        boxShadow: transfer.isStuck ? '0 0 12px rgba(239,68,68,0.08)' : 'none',
      }}
    >
      <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: 6 }}>
        <span style={{ fontSize: 13, fontWeight: 600, color: colors.text }}>{transfer.client.name}</span>
        <AccountTag type={transfer.accountType} />
      </div>
      <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: 4 }}>
        <span style={{ fontSize: 12, color: colors.textMuted }}>From {transfer.sourceInstitution}</span>
        <span style={{ fontSize: 13, fontWeight: 600, color: colors.text }}>
          ${transfer.amount.toLocaleString()}
        </span>
      </div>
      <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
        <span style={{ fontSize: 11, color: colors.textDim }}>Initiated {transfer.initiatedAt}</span>
        {transfer.isStuck ? (
          <span style={{ fontSize: 11, fontWeight: 600, color: colors.red }}>
            ⚠ Stuck {transfer.daysInCurrentStage}d
          </span>
        ) : (
          <span style={{ fontSize: 11, color: colors.textDim }}>
            {transfer.daysInCurrentStage}d in stage
          </span>
        )}
      </div>
    </div>
  )
}
