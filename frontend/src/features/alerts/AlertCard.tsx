import { useState } from 'react'
import { colors, severityColors } from '../../styles/theme'
import { Badge } from '../../components/Badge'
import { SeverityDot } from '../../components/SeverityDot'
import { IconButton } from '../../components/IconButton'
import {
  useSendAlertMutation,
  useTrackAlertMutation,
  useSnoozeAlertMutation,
  useAcknowledgeAlertMutation,
} from '../../generated/graphql'
import type { GetAlertsQuery } from '../../generated/graphql'

type AlertData = GetAlertsQuery['alerts'][number]

interface AlertCardProps {
  alert: AlertData
  onSnoozed?: (alertId: string) => void
}

function formatTime(dateStr: string): string {
  const date = new Date(dateStr)
  const now = new Date()
  const diffMs = now.getTime() - date.getTime()
  const diffHours = Math.floor(diffMs / (1000 * 60 * 60))
  const diffDays = Math.floor(diffMs / (1000 * 60 * 60 * 24))
  if (diffHours < 1) return 'Just now'
  if (diffHours < 24) return `${diffHours}h ago`
  if (diffDays === 1) return '1 day ago'
  return `${diffDays} days ago`
}

export function AlertCard({ alert, onSnoozed }: AlertCardProps) {
  const [expanded, setExpanded] = useState(false)
  const [draftText, setDraftText] = useState(alert.draftMessage || '')
  const [actionTaken, setActionTaken] = useState<'sent' | 'tracked' | null>(null)
  const [error, setError] = useState<string | null>(null)

  const [sendAlert, { loading: sendLoading }] = useSendAlertMutation()
  const [trackAlert, { loading: trackLoading }] = useTrackAlertMutation()
  const [snoozeAlert, { loading: snoozeLoading }] = useSnoozeAlertMutation()
  const [acknowledgeAlert, { loading: ackLoading }] = useAcknowledgeAlertMutation()

  const sc = severityColors[alert.severity] || severityColors['INFO']
  const isInfo = alert.severity === 'INFO'
  const isClosed = alert.status === 'CLOSED'
  const isActed = alert.status === 'ACTED' || actionTaken !== null
  const isDimmed = isInfo || isClosed || isActed

  const showActions = !isClosed && !isActed && !isInfo
  const mutationInFlight = sendLoading || trackLoading || snoozeLoading || ackLoading

  const handleSend = async () => {
    setError(null)
    try {
      await sendAlert({ variables: { alertId: alert.id, message: draftText || null } })
      setActionTaken('sent')
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to send alert')
    }
  }

  const handleTrack = async () => {
    setError(null)
    try {
      await trackAlert({ variables: { alertId: alert.id, actionItemText: alert.summary } })
      setActionTaken('tracked')
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to track alert')
    }
  }

  const handleSnooze = async () => {
    setError(null)
    try {
      await snoozeAlert({ variables: { alertId: alert.id, until: null } })
      onSnoozed?.(alert.id)
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to snooze alert')
    }
  }

  const handleAcknowledge = async () => {
    setError(null)
    try {
      await acknowledgeAlert({ variables: { alertId: alert.id } })
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to acknowledge alert')
    }
  }

  const getBadge = () => {
    if (isClosed) return <Badge color={colors.textMuted} bgColor={colors.infoDim}>Resolved</Badge>
    if (actionTaken === 'sent' || isInfo || alert.status === 'ACTED')
      return <Badge color={colors.green} bgColor={colors.greenDim}>Sent ✓</Badge>
    if (actionTaken === 'tracked')
      return <Badge color={colors.green} bgColor={colors.greenDim}>Tracked ✓</Badge>
    return null
  }

  return (
    <div
      style={{
        background: colors.surface,
        border: `1px solid ${colors.border}`,
        borderLeft: `3px solid ${sc.dot}`,
        borderRadius: 8,
        padding: '14px 16px',
        marginBottom: 8,
        opacity: isDimmed ? 0.65 : 1,
        transition: 'all 0.2s',
      }}
    >
      <div style={{ display: 'flex', gap: 12 }}>
        <SeverityDot severity={alert.severity} />
        <div style={{ flex: 1, minWidth: 0 }}>
          <div style={{ display: 'flex', alignItems: 'center', gap: 8, marginBottom: 4, flexWrap: 'wrap' }}>
            <span style={{ fontSize: 13, fontWeight: 600, color: colors.text }}>{alert.client.name}</span>
            <Badge color={sc.text} bgColor={sc.bg}>{alert.category}</Badge>
            {getBadge()}
            <span style={{ fontSize: 11, color: colors.textDim, marginLeft: 'auto', flexShrink: 0 }}>
              {formatTime(alert.createdAt)}
            </span>
          </div>
          <p style={{ fontSize: 13, lineHeight: 1.55, color: colors.textMuted, margin: 0 }}>{alert.summary}</p>

          {alert.draftMessage && !isDimmed && (
            <button
              onClick={() => setExpanded(!expanded)}
              style={{
                marginTop: 8,
                fontSize: 12,
                color: colors.accent,
                background: 'none',
                border: 'none',
                cursor: 'pointer',
                padding: 0,
                fontFamily: 'inherit',
              }}
            >
              {expanded ? 'Hide draft ▴' : 'Preview draft ▾'}
            </button>
          )}

          {expanded && alert.draftMessage && (
            <textarea
              value={draftText}
              onChange={(e) => setDraftText(e.target.value)}
              style={{
                marginTop: 8,
                padding: '10px 12px',
                background: colors.surfaceRaised,
                borderRadius: 6,
                border: `1px solid ${colors.borderLight}`,
                fontSize: 12.5,
                lineHeight: 1.6,
                color: colors.textMuted,
                fontFamily: 'inherit',
                width: '100%',
                minHeight: 80,
                resize: 'vertical',
                outline: 'none',
                boxSizing: 'border-box',
              }}
            />
          )}

          {error && (
            <p style={{ fontSize: 12, color: colors.red, margin: '6px 0 0' }}>{error}</p>
          )}
        </div>

        {showActions && (
          <div style={{ display: 'flex', flexDirection: 'column', gap: 4, flexShrink: 0 }}>
            <IconButton variant="send" title="Send to client" onClick={handleSend} disabled={mutationInFlight}>
              <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.5" strokeLinecap="round" strokeLinejoin="round"><path d="M22 2L11 13"/><path d="M22 2L15 22L11 13L2 9L22 2Z"/></svg>
            </IconButton>
            <IconButton title="Track" onClick={handleTrack} disabled={mutationInFlight}>
              <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><path d="M12 20h9"/><path d="M16.5 3.5a2.121 2.121 0 013 3L7 19l-4 1 1-4L16.5 3.5z"/></svg>
            </IconButton>
            <IconButton variant="dismiss" title="Snooze" onClick={handleSnooze} disabled={mutationInFlight}>
              <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><circle cx="12" cy="12" r="10"/><path d="M12 6v6l4 2"/></svg>
            </IconButton>
          </div>
        )}

        {isInfo && !isClosed && !isActed && (
          <div style={{ display: 'flex', flexDirection: 'column', gap: 4, flexShrink: 0 }}>
            <IconButton title="Acknowledge" onClick={handleAcknowledge} disabled={ackLoading}>
              <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.5" strokeLinecap="round" strokeLinejoin="round"><path d="M20 6L9 17l-5-5"/></svg>
            </IconButton>
          </div>
        )}
      </div>
    </div>
  )
}
