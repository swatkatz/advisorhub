import { useState } from 'react'
import { colors, severityOrder } from '../../styles/theme'
import { AlertCard } from './AlertCard'
import {
  useGetAlertsQuery,
  useAlertFeedSubscription,
  useRunMorningSweepMutation,
} from '../../generated/graphql'
import type { GetAlertsQuery, AlertFilter } from '../../generated/graphql'

type AlertData = GetAlertsQuery['alerts'][number]

const FILTERS = [
  { key: 'all', label: 'All' },
  { key: 'attention', label: 'Needs attention' },
  { key: 'critical', label: 'Critical' },
  { key: 'urgent', label: 'Urgent' },
  { key: 'advisory', label: 'Advisory' },
] as const

function buildFilter(key: string): AlertFilter | undefined {
  switch (key) {
    case 'attention': return { status: 'OPEN' }
    case 'critical': return { severity: 'CRITICAL' }
    case 'urgent': return { severity: 'URGENT' }
    case 'advisory': return { severity: 'ADVISORY' }
    default: return undefined
  }
}

function sortAlerts(alerts: AlertData[]): AlertData[] {
  return [...alerts].sort((a, b) => {
    const sA = severityOrder[a.severity] ?? 99
    const sB = severityOrder[b.severity] ?? 99
    if (sA !== sB) return sA - sB
    return new Date(b.createdAt).getTime() - new Date(a.createdAt).getTime()
  })
}

export function AlertFeed() {
  const [filter, setFilter] = useState('all')
  const [snoozedIds, setSnoozedIds] = useState<Set<string>>(new Set())
  const [sweepResult, setSweepResult] = useState<{ alertsGenerated: number; alertsUpdated: number; alertsSkipped: number; duration: string } | null>(null)

  const filterInput = buildFilter(filter)
  const { data, loading, error, refetch } = useGetAlertsQuery({
    variables: { advisorId: 'adv1', filter: filterInput },
  })

  const [runSweep, { loading: sweepLoading }] = useRunMorningSweepMutation()

  useAlertFeedSubscription({
    variables: { advisorId: 'adv1' },
    onData: () => {
      refetch()
    },
  })

  const handleSweep = async () => {
    setSweepResult(null)
    try {
      const result = await runSweep({ variables: { advisorId: 'adv1' } })
      if (result.data) {
        setSweepResult(result.data.runMorningSweep)
      }
      refetch()
    } catch {
      // Error handled by Apollo
    }
  }

  const alerts = data?.alerts ?? []
  const visibleAlerts = sortAlerts(
    alerts.filter((a) => !snoozedIds.has(a.id) && a.status !== 'SNOOZED')
  )
  const unresolvedCount = alerts.filter((a) => a.status !== 'CLOSED' && a.status !== 'SNOOZED').length

  return (
    <div style={{ height: '100%', display: 'flex', flexDirection: 'column' }}>
      <div style={{ padding: '20px 20px 0' }}>
        <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: 16 }}>
          <div>
            <h2 style={{ fontSize: 18, fontWeight: 700, color: colors.text, margin: 0 }}>Alert Feed</h2>
            <p style={{ fontSize: 12, color: colors.textDim, margin: '4px 0 0' }}>
              {unresolvedCount} alerts need your attention
            </p>
          </div>
          <button
            onClick={handleSweep}
            disabled={sweepLoading}
            style={{
              display: 'flex',
              alignItems: 'center',
              gap: 6,
              padding: '8px 14px',
              borderRadius: 6,
              background: sweepLoading
                ? colors.surfaceRaised
                : 'linear-gradient(135deg, rgba(34,211,238,0.15), rgba(129,140,248,0.15))',
              border: `1px solid ${sweepLoading ? colors.borderLight : 'rgba(34,211,238,0.3)'}`,
              color: sweepLoading ? colors.textDim : colors.accent,
              fontSize: 12,
              fontWeight: 600,
              cursor: sweepLoading ? 'default' : 'pointer',
              fontFamily: 'inherit',
            }}
          >
            {sweepLoading ? (
              <>
                <span style={{ display: 'inline-block', animation: 'spin 1s linear infinite', fontSize: 13 }}>⟳</span>
                {' '}Scanning...
              </>
            ) : (
              <>
                <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.5">
                  <path d="M21 12a9 9 0 11-6.22-8.56" />
                  <path d="M21 3v9h-9" />
                </svg>
                {' '}Run morning sweep
              </>
            )}
          </button>
        </div>

        {sweepResult && (
          <div
            style={{
              padding: '10px 14px',
              background: colors.surfaceRaised,
              borderRadius: 6,
              border: `1px solid ${colors.borderLight}`,
              marginBottom: 12,
              fontSize: 12,
              color: colors.textMuted,
            }}
          >
            Sweep complete: {sweepResult.alertsGenerated} generated, {sweepResult.alertsUpdated} updated,{' '}
            {sweepResult.alertsSkipped} skipped ({sweepResult.duration})
          </div>
        )}

        <div style={{ display: 'flex', gap: 6, marginBottom: 14 }}>
          {FILTERS.map((f) => (
            <button
              key={f.key}
              onClick={() => setFilter(f.key)}
              style={{
                padding: '5px 10px',
                borderRadius: 5,
                fontSize: 11,
                fontWeight: 500,
                cursor: 'pointer',
                fontFamily: 'inherit',
                transition: 'all 0.15s',
                background: filter === f.key ? colors.surfaceRaised : 'transparent',
                border: `1px solid ${filter === f.key ? colors.borderLight : 'transparent'}`,
                color: filter === f.key ? colors.text : colors.textDim,
              }}
            >
              {f.label}
            </button>
          ))}
        </div>
      </div>

      <div style={{ flex: 1, overflow: 'auto', padding: '0 20px 20px' }}>
        {loading && (
          <div style={{ padding: 40, textAlign: 'center', color: colors.textDim, fontSize: 13 }}>
            Loading alerts...
          </div>
        )}
        {error && (
          <div style={{ padding: 20, color: colors.red, fontSize: 13 }}>
            Error loading alerts: {error.message}
          </div>
        )}
        {!loading && !error && visibleAlerts.length === 0 && (
          <div style={{ padding: 40, textAlign: 'center', color: colors.textDim, fontSize: 13 }}>
            No alerts to show
          </div>
        )}
        {visibleAlerts.map((alert) => (
          <AlertCard
            key={alert.id}
            alert={alert}
            onSnoozed={(id) => setSnoozedIds((prev) => new Set(prev).add(id))}
          />
        ))}
      </div>
    </div>
  )
}
