import { useState } from 'react'
import { colors, actionItemStatusColors, actionItemStatusLabels } from '../../styles/theme'
import { Avatar } from '../../components/Avatar'
import { AccountTag } from '../../components/AccountTag'
import { ContributionBar } from './ContributionBar'
import { GoalBar } from './GoalBar'
import {
  useGetClientQuery,
  useGetContributionSummaryQuery,
  useAddNoteMutation,
} from '../../generated/graphql'

interface ClientDetailProps {
  clientId: string
}

function getInitials(name: string): string {
  const parts = name.split(/[\s&]+/).filter(Boolean)
  if (parts.length >= 2) {
    return (parts[0][0] + parts[1][0]).toUpperCase()
  }
  return name.substring(0, 2).toUpperCase()
}

export function ClientDetail({ clientId }: ClientDetailProps) {
  const [noteText, setNoteText] = useState('')
  const [showNoteInput, setShowNoteInput] = useState(false)

  const { data: clientData, loading: clientLoading, error: clientError } = useGetClientQuery({
    variables: { id: clientId },
  })

  const { data: contribData } = useGetContributionSummaryQuery({
    variables: { clientId, taxYear: 2026 },
  })

  const [addNote, { loading: noteLoading }] = useAddNoteMutation()

  if (clientLoading) {
    return (
      <div style={{ padding: 30, textAlign: 'center', color: colors.textDim, fontSize: 13 }}>
        Loading client details...
      </div>
    )
  }

  if (clientError || !clientData) {
    return (
      <div style={{ padding: 30, textAlign: 'center', color: colors.red, fontSize: 13 }}>
        {clientError ? `Error: ${clientError.message}` : 'Client not found'}
      </div>
    )
  }

  const client = clientData.client
  const contributions = contribData?.contributionSummary.accounts ?? []
  const externalAccounts = client.externalAccounts ?? []

  const handleAddNote = async () => {
    if (!noteText.trim()) return
    try {
      await addNote({
        variables: { clientId, text: noteText.trim() },
        refetchQueries: ['GetClient'],
      })
      setNoteText('')
      setShowNoteInput(false)
    } catch {
      // Error handled by Apollo
    }
  }

  return (
    <div style={{ padding: 20, overflow: 'auto', height: '100%' }}>
      {/* Header */}
      <div style={{ display: 'flex', alignItems: 'center', gap: 12, marginBottom: 20 }}>
        <Avatar initials={getInitials(client.name)} size={44} />
        <div>
          <h3 style={{ fontSize: 16, fontWeight: 700, color: colors.text, margin: 0 }}>{client.name}</h3>
          <p style={{ fontSize: 12, color: colors.textDim, margin: '2px 0 0' }}>
            AUM: ${client.aum.toLocaleString()} · Last meeting: {client.lastMeeting}
          </p>
        </div>
      </div>

      {/* Contribution Summary */}
      {contributions.length > 0 && (
        <div style={{ marginBottom: 20 }}>
          <h4
            style={{
              fontSize: 13,
              fontWeight: 600,
              color: colors.textMuted,
              letterSpacing: '0.04em',
              textTransform: 'uppercase',
              margin: '0 0 12px',
              borderBottom: `1px solid ${colors.border}`,
              paddingBottom: 8,
            }}
          >
            Contribution Room — 2026
          </h4>
          {contributions.map((c) => (
            <ContributionBar key={c.accountType} contribution={c} />
          ))}
        </div>
      )}

      {/* External Accounts */}
      {externalAccounts.length > 0 && (
        <div style={{ marginBottom: 20 }}>
          <h4
            style={{
              fontSize: 13,
              fontWeight: 600,
              color: colors.textMuted,
              letterSpacing: '0.04em',
              textTransform: 'uppercase',
              margin: '0 0 10px',
              borderBottom: `1px solid ${colors.border}`,
              paddingBottom: 8,
            }}
          >
            External Accounts
          </h4>
          {externalAccounts.map((ea) => (
            <div
              key={ea.id}
              style={{
                display: 'flex',
                justifyContent: 'space-between',
                alignItems: 'center',
                padding: '6px 0',
                borderBottom: `1px solid ${colors.border}`,
              }}
            >
              <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
                <span style={{ fontSize: 12, color: colors.text }}>{ea.institution}</span>
                <AccountTag type={ea.accountType} />
              </div>
              <span style={{ fontSize: 13, fontWeight: 600, color: colors.text }}>
                ${ea.balance.toLocaleString()}
              </span>
            </div>
          ))}
        </div>
      )}

      {/* Goals */}
      {client.goals.length > 0 && (
        <div style={{ marginBottom: 20 }}>
          <h4
            style={{
              fontSize: 13,
              fontWeight: 600,
              color: colors.textMuted,
              letterSpacing: '0.04em',
              textTransform: 'uppercase',
              margin: '0 0 12px',
              borderBottom: `1px solid ${colors.border}`,
              paddingBottom: 8,
            }}
          >
            Goals
          </h4>
          {client.goals.map((g) => (
            <GoalBar key={g.id} goal={g} />
          ))}
        </div>
      )}

      {/* Action Items */}
      {client.actionItems.length > 0 && (
        <div style={{ marginBottom: 20 }}>
          <h4
            style={{
              fontSize: 13,
              fontWeight: 600,
              color: colors.textMuted,
              letterSpacing: '0.04em',
              textTransform: 'uppercase',
              margin: '0 0 10px',
              borderBottom: `1px solid ${colors.border}`,
              paddingBottom: 8,
            }}
          >
            Action Items
          </h4>
          {client.actionItems.map((ai) => (
            <div
              key={ai.id}
              style={{
                display: 'flex',
                alignItems: 'flex-start',
                gap: 8,
                padding: '8px 0',
                borderBottom: `1px solid ${colors.border}`,
              }}
            >
              <div
                style={{
                  width: 6,
                  height: 6,
                  borderRadius: '50%',
                  background: actionItemStatusColors[ai.status] || colors.textDim,
                  marginTop: 5,
                  flexShrink: 0,
                }}
              />
              <div style={{ flex: 1 }}>
                <span style={{ fontSize: 12.5, color: colors.text }}>{ai.text}</span>
                <div style={{ display: 'flex', gap: 8, marginTop: 3 }}>
                  <span
                    style={{
                      fontSize: 10,
                      color: actionItemStatusColors[ai.status] || colors.textDim,
                      fontWeight: 600,
                    }}
                  >
                    {actionItemStatusLabels[ai.status] || ai.status}
                  </span>
                  {ai.dueDate && (
                    <span style={{ fontSize: 10, color: colors.textDim }}>Due: {ai.dueDate}</span>
                  )}
                </div>
              </div>
            </div>
          ))}
        </div>
      )}

      {/* Notes */}
      <div>
        <h4
          style={{
            fontSize: 13,
            fontWeight: 600,
            color: colors.textMuted,
            letterSpacing: '0.04em',
            textTransform: 'uppercase',
            margin: '0 0 10px',
            borderBottom: `1px solid ${colors.border}`,
            paddingBottom: 8,
          }}
        >
          Advisor Notes
        </h4>
        {client.notes
          .slice()
          .sort((a, b) => new Date(b.date).getTime() - new Date(a.date).getTime())
          .map((n) => (
            <div
              key={n.id}
              style={{
                padding: '8px 0',
                borderBottom: `1px solid ${colors.border}`,
              }}
            >
              <span style={{ fontSize: 10, fontWeight: 600, color: colors.textDim }}>{n.date}</span>
              <p style={{ fontSize: 12.5, color: colors.textMuted, margin: '4px 0 0', lineHeight: 1.5 }}>
                {n.text}
              </p>
            </div>
          ))}

        {showNoteInput ? (
          <div style={{ marginTop: 10 }}>
            <textarea
              value={noteText}
              onChange={(e) => setNoteText(e.target.value)}
              placeholder="Write a note..."
              style={{
                width: '100%',
                padding: '8px 12px',
                borderRadius: 6,
                fontSize: 12,
                fontFamily: 'inherit',
                background: colors.surfaceRaised,
                border: `1px solid ${colors.border}`,
                color: colors.text,
                outline: 'none',
                minHeight: 60,
                resize: 'vertical',
                boxSizing: 'border-box',
              }}
            />
            <div style={{ display: 'flex', gap: 6, marginTop: 6 }}>
              <button
                onClick={handleAddNote}
                disabled={noteLoading || !noteText.trim()}
                style={{
                  padding: '6px 12px',
                  borderRadius: 6,
                  fontSize: 12,
                  fontWeight: 600,
                  background: colors.accentDim,
                  border: `1px solid rgba(34,211,238,0.3)`,
                  color: colors.accent,
                  cursor: noteLoading ? 'default' : 'pointer',
                  fontFamily: 'inherit',
                }}
              >
                {noteLoading ? 'Saving...' : 'Save note'}
              </button>
              <button
                onClick={() => { setShowNoteInput(false); setNoteText('') }}
                style={{
                  padding: '6px 12px',
                  borderRadius: 6,
                  fontSize: 12,
                  background: 'transparent',
                  border: `1px solid ${colors.borderLight}`,
                  color: colors.textDim,
                  cursor: 'pointer',
                  fontFamily: 'inherit',
                }}
              >
                Cancel
              </button>
            </div>
          </div>
        ) : (
          <button
            onClick={() => setShowNoteInput(true)}
            style={{
              marginTop: 10,
              width: '100%',
              padding: 8,
              borderRadius: 6,
              fontSize: 12,
              fontWeight: 500,
              background: 'transparent',
              border: `1px dashed ${colors.borderLight}`,
              color: colors.textDim,
              cursor: 'pointer',
              fontFamily: 'inherit',
            }}
          >
            + Add note
          </button>
        )}
      </div>
    </div>
  )
}
