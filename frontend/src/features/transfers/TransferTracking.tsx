import { colors } from '../../styles/theme'
import { TransferCard } from './TransferCard'
import { useGetTransfersQuery } from '../../generated/graphql'
import type { TransferStatus } from '../../generated/graphql'

const TRANSFER_STAGES: { key: TransferStatus; label: string }[] = [
  { key: 'INITIATED', label: 'Initiated' },
  { key: 'DOCUMENTS_SUBMITTED', label: 'Documents Submitted' },
  { key: 'IN_REVIEW', label: 'In Review' },
  { key: 'IN_TRANSIT', label: 'In Transit' },
  { key: 'RECEIVED', label: 'Received' },
  { key: 'INVESTED', label: 'Invested' },
]

export function TransferTracking() {
  const { data, loading, error } = useGetTransfersQuery({
    variables: { advisorId: 'adv1' },
  })

  const transfers = data?.transfers ?? []

  return (
    <div style={{ height: '100%', display: 'flex', flexDirection: 'column' }}>
      <div style={{ padding: '20px 20px 0' }}>
        <h2 style={{ fontSize: 18, fontWeight: 700, color: colors.text, margin: 0 }}>Transfer Tracking</h2>
        <p style={{ fontSize: 12, color: colors.textDim, margin: '4px 0 16px' }}>
          {transfers.length} active transfers across clients
        </p>
      </div>

      {loading && (
        <div style={{ padding: 40, textAlign: 'center', color: colors.textDim, fontSize: 13 }}>
          Loading transfers...
        </div>
      )}
      {error && (
        <div style={{ padding: 20, color: colors.red, fontSize: 13 }}>
          Error loading transfers: {error.message}
        </div>
      )}

      {!loading && !error && (
        <div style={{ flex: 1, overflow: 'auto', padding: '0 20px 20px' }}>
          <div style={{ display: 'flex', gap: 12, minWidth: 'max-content', height: '100%' }}>
            {TRANSFER_STAGES.map((stage) => {
              const stageTransfers = transfers.filter((t) => t.status === stage.key)
              return (
                <div key={stage.key} style={{ width: 220, flexShrink: 0 }}>
                  <div style={{ display: 'flex', alignItems: 'center', gap: 6, marginBottom: 10, padding: '0 2px' }}>
                    <span style={{ fontSize: 12, fontWeight: 600, color: colors.textMuted, letterSpacing: '0.02em' }}>
                      {stage.label}
                    </span>
                    {stageTransfers.length > 0 && (
                      <span
                        style={{
                          fontSize: 10,
                          fontWeight: 700,
                          color: colors.textDim,
                          background: colors.surfaceRaised,
                          padding: '2px 6px',
                          borderRadius: 10,
                        }}
                      >
                        {stageTransfers.length}
                      </span>
                    )}
                  </div>
                  <div style={{ minHeight: 80, background: 'rgba(255,255,255,0.015)', borderRadius: 8, padding: 6 }}>
                    {stageTransfers.map((t) => (
                      <TransferCard key={t.id} transfer={t} />
                    ))}
                    {stageTransfers.length === 0 && (
                      <div style={{ padding: 20, textAlign: 'center', fontSize: 11, color: colors.textDim }}>—</div>
                    )}
                  </div>
                </div>
              )
            })}
          </div>
        </div>
      )}
    </div>
  )
}
