import { useState } from 'react'
import { colors } from './styles/theme'
import { Avatar } from './components/Avatar'
import { AlertFeed } from './features/alerts/AlertFeed'
import { TransferTracking } from './features/transfers/TransferTracking'
import { ClientBook } from './features/clients/ClientBook'
import { useGetAdvisorQuery, useGetAlertsQuery } from './generated/graphql'

type Section = 'alerts' | 'transfers' | 'clients'

const NAV_ITEMS: { key: Section; label: string; icon: React.ReactNode }[] = [
  {
    key: 'alerts',
    label: 'Alert Feed',
    icon: (
      <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
        <path d="M18 8A6 6 0 006 8c0 7-3 9-3 9h18s-3-2-3-9" />
        <path d="M13.73 21a2 2 0 01-3.46 0" />
      </svg>
    ),
  },
  {
    key: 'transfers',
    label: 'Transfers',
    icon: (
      <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
        <path d="M17 1l4 4-4 4" />
        <path d="M3 11V9a4 4 0 014-4h14" />
        <path d="M7 23l-4-4 4-4" />
        <path d="M21 13v2a4 4 0 01-4 4H3" />
      </svg>
    ),
  },
  {
    key: 'clients',
    label: 'Client Book',
    icon: (
      <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
        <path d="M17 21v-2a4 4 0 00-4-4H5a4 4 0 00-4 4v2" />
        <circle cx="9" cy="7" r="4" />
        <path d="M23 21v-2a4 4 0 00-3-3.87" />
        <path d="M16 3.13a4 4 0 010 7.75" />
      </svg>
    ),
  },
]

function getInitials(name: string): string {
  const parts = name.split(/[\s&]+/).filter(Boolean)
  if (parts.length >= 2) {
    return (parts[0][0] + parts[parts.length - 1][0]).toUpperCase()
  }
  return name.substring(0, 2).toUpperCase()
}

export default function App() {
  const [activeSection, setActiveSection] = useState<Section>('alerts')

  const { data: advisorData } = useGetAdvisorQuery({
    variables: { id: 'adv1' },
  })

  const { data: alertsData } = useGetAlertsQuery({
    variables: { advisorId: 'adv1' },
  })

  const advisor = advisorData?.advisor
  const unresolvedCount = (alertsData?.alerts ?? []).filter(
    (a) => a.status !== 'CLOSED',
  ).length

  return (
    <div
      style={{
        display: 'flex',
        height: '100vh',
        width: '100vw',
        background: colors.bg,
        fontFamily: "'DM Sans', 'Segoe UI', system-ui, sans-serif",
        color: colors.text,
        overflow: 'hidden',
      }}
    >
      {/* Sidebar */}
      <div
        style={{
          width: 220,
          background: colors.surface,
          borderRight: `1px solid ${colors.border}`,
          display: 'flex',
          flexDirection: 'column',
          flexShrink: 0,
        }}
      >
        {/* Logo */}
        <div style={{ padding: '20px 16px', borderBottom: `1px solid ${colors.border}` }}>
          <div style={{ display: 'flex', alignItems: 'center', gap: 10 }}>
            <div
              style={{
                width: 32,
                height: 32,
                borderRadius: 8,
                background: `linear-gradient(135deg, ${colors.accent}, #818CF8)`,
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'center',
              }}
            >
              <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="white" strokeWidth="2.5" strokeLinecap="round" strokeLinejoin="round">
                <path d="M12 2L2 7l10 5 10-5-10-5z" />
                <path d="M2 17l10 5 10-5" />
                <path d="M2 12l10 5 10-5" />
              </svg>
            </div>
            <div>
              <div style={{ fontSize: 14, fontWeight: 700, color: colors.text, letterSpacing: '-0.01em' }}>
                AdvisorHub
              </div>
              <div style={{ fontSize: 10, color: colors.textDim }}>by Wealthsimple</div>
            </div>
          </div>
        </div>

        {/* Advisor info */}
        <div style={{ padding: '14px 16px', borderBottom: `1px solid ${colors.border}` }}>
          <div style={{ display: 'flex', alignItems: 'center', gap: 10 }}>
            <Avatar initials={advisor ? getInitials(advisor.name) : '...'} size={32} />
            <div>
              <div style={{ fontSize: 13, fontWeight: 600, color: colors.text }}>
                {advisor?.name ?? 'Loading...'}
              </div>
              <div style={{ fontSize: 10, color: colors.textDim }}>
                {advisor?.role ?? ''}
              </div>
            </div>
          </div>
        </div>

        {/* Nav */}
        <nav style={{ padding: '12px 8px', flex: 1 }}>
          {NAV_ITEMS.map((item) => {
            const isActive = activeSection === item.key
            return (
              <button
                key={item.key}
                onClick={() => setActiveSection(item.key)}
                style={{
                  display: 'flex',
                  alignItems: 'center',
                  gap: 10,
                  width: '100%',
                  padding: '10px 12px',
                  borderRadius: 8,
                  border: 'none',
                  cursor: 'pointer',
                  marginBottom: 2,
                  fontFamily: 'inherit',
                  background: isActive ? colors.surfaceRaised : 'transparent',
                  color: isActive ? colors.text : colors.textMuted,
                  transition: 'all 0.15s',
                  position: 'relative',
                  fontSize: 13,
                  fontWeight: isActive ? 600 : 400,
                  textAlign: 'left',
                }}
              >
                {isActive && (
                  <div
                    style={{
                      position: 'absolute',
                      left: 0,
                      top: '50%',
                      transform: 'translateY(-50%)',
                      width: 3,
                      height: 20,
                      borderRadius: 2,
                      background: colors.accent,
                    }}
                  />
                )}
                {item.icon}
                <span>{item.label}</span>
                {item.key === 'alerts' && unresolvedCount > 0 && (
                  <span
                    style={{
                      marginLeft: 'auto',
                      fontSize: 10,
                      fontWeight: 700,
                      color: '#fff',
                      background: colors.critical,
                      padding: '2px 6px',
                      borderRadius: 10,
                      minWidth: 18,
                      textAlign: 'center',
                    }}
                  >
                    {unresolvedCount}
                  </span>
                )}
              </button>
            )
          })}
        </nav>
      </div>

      {/* Main content */}
      <div style={{ flex: 1, overflow: 'hidden' }}>
        {activeSection === 'alerts' && <AlertFeed />}
        {activeSection === 'transfers' && <TransferTracking />}
        {activeSection === 'clients' && <ClientBook />}
      </div>
    </div>
  )
}
