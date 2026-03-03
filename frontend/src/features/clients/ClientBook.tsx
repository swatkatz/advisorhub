import { useState } from 'react'
import { colors, healthColors } from '../../styles/theme'
import { Avatar } from '../../components/Avatar'
import { AccountTag } from '../../components/AccountTag'
import { ClientDetail } from './ClientDetail'
import { useGetClientsQuery } from '../../generated/graphql'

function getInitials(name: string): string {
  const parts = name.split(/[\s&]+/).filter(Boolean)
  if (parts.length >= 2) {
    return (parts[0][0] + parts[1][0]).toUpperCase()
  }
  return name.substring(0, 2).toUpperCase()
}

export function ClientBook() {
  const [selectedClientId, setSelectedClientId] = useState<string | null>(null)
  const [searchQuery, setSearchQuery] = useState('')

  const { data, loading, error } = useGetClientsQuery({
    variables: { advisorId: 'adv1' },
  })

  const clients = data?.clients ?? []
  const filtered = clients.filter((c) =>
    c.name.toLowerCase().includes(searchQuery.toLowerCase()),
  )
  const totalAum = clients.reduce((sum, c) => sum + c.aum, 0)

  const handleRowClick = (clientId: string) => {
    setSelectedClientId(selectedClientId === clientId ? null : clientId)
  }

  return (
    <div style={{ height: '100%', display: 'flex' }}>
      {/* Client list */}
      <div
        style={{
          width: selectedClientId ? 340 : '100%',
          borderRight: selectedClientId ? `1px solid ${colors.border}` : 'none',
          display: 'flex',
          flexDirection: 'column',
          transition: 'width 0.2s',
          minWidth: selectedClientId ? 340 : undefined,
        }}
      >
        <div style={{ padding: '20px 20px 0' }}>
          <h2 style={{ fontSize: 18, fontWeight: 700, color: colors.text, margin: '0 0 4px' }}>Client Book</h2>
          <p style={{ fontSize: 12, color: colors.textDim, margin: '0 0 14px' }}>
            {clients.length} clients · ${(totalAum / 1000000).toFixed(1)}M AUM
          </p>
          <input
            type="text"
            placeholder="Search clients..."
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
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
              marginBottom: 12,
              boxSizing: 'border-box',
            }}
          />
        </div>

        <div style={{ flex: 1, overflow: 'auto', padding: '0 20px 20px' }}>
          {loading && (
            <div style={{ padding: 40, textAlign: 'center', color: colors.textDim, fontSize: 13 }}>
              Loading clients...
            </div>
          )}
          {error && (
            <div style={{ padding: 20, color: colors.red, fontSize: 13 }}>
              Error loading clients: {error.message}
            </div>
          )}

          {!loading && !error && (
            <>
              {/* Table header */}
              <div
                style={{
                  display: 'grid',
                  gridTemplateColumns: selectedClientId
                    ? '1fr 50px'
                    : '1fr 100px 90px 80px 50px',
                  gap: 8,
                  padding: '6px 0',
                  borderBottom: `1px solid ${colors.border}`,
                  marginBottom: 4,
                }}
              >
                <span style={{ fontSize: 10, fontWeight: 600, color: colors.textDim, letterSpacing: '0.05em', textTransform: 'uppercase' }}>
                  Client
                </span>
                {!selectedClientId && (
                  <>
                    <span style={{ fontSize: 10, fontWeight: 600, color: colors.textDim, letterSpacing: '0.05em', textTransform: 'uppercase' }}>Accounts</span>
                    <span style={{ fontSize: 10, fontWeight: 600, color: colors.textDim, letterSpacing: '0.05em', textTransform: 'uppercase' }}>AUM</span>
                    <span style={{ fontSize: 10, fontWeight: 600, color: colors.textDim, letterSpacing: '0.05em', textTransform: 'uppercase' }}>Last met</span>
                  </>
                )}
                <span style={{ fontSize: 10, fontWeight: 600, color: colors.textDim, letterSpacing: '0.05em', textTransform: 'uppercase' }}></span>
              </div>

              {filtered.map((client) => {
                const isSelected = selectedClientId === client.id
                return (
                  <div
                    key={client.id}
                    onClick={() => handleRowClick(client.id)}
                    style={{
                      display: 'grid',
                      gridTemplateColumns: selectedClientId
                        ? '1fr 50px'
                        : '1fr 100px 90px 80px 50px',
                      gap: 8,
                      padding: '10px 0',
                      borderBottom: `1px solid ${colors.border}`,
                      cursor: 'pointer',
                      alignItems: 'center',
                      background: isSelected ? colors.surfaceRaised : 'transparent',
                      borderRadius: isSelected ? 6 : 0,
                      marginLeft: isSelected ? -8 : 0,
                      marginRight: isSelected ? -8 : 0,
                      paddingLeft: isSelected ? 8 : 0,
                      paddingRight: isSelected ? 8 : 0,
                    }}
                  >
                    <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
                      <Avatar initials={getInitials(client.name)} size={28} />
                      <span style={{ fontSize: 13, fontWeight: 500, color: colors.text }}>{client.name}</span>
                      <span
                        style={{
                          width: 7,
                          height: 7,
                          borderRadius: '50%',
                          background: healthColors[client.health] || colors.textDim,
                          display: 'inline-block',
                        }}
                      />
                    </div>
                    {!selectedClientId && (
                      <>
                        <div style={{ display: 'flex', gap: 3, flexWrap: 'wrap' }}>
                          {client.accounts.map((a) => (
                            <AccountTag key={a.id} type={a.accountType} />
                          ))}
                        </div>
                        <span style={{ fontSize: 12, color: colors.text, fontWeight: 500 }}>
                          ${(client.aum / 1000).toFixed(0)}K
                        </span>
                        <span style={{ fontSize: 11, color: colors.textDim }}>
                          {client.lastMeeting}
                        </span>
                      </>
                    )}
                    <div>
                      {client.alerts.length > 0 && (
                        <span
                          style={{
                            fontSize: 10,
                            fontWeight: 700,
                            color: colors.bg,
                            background:
                              client.alerts.length >= 3
                                ? colors.red
                                : client.alerts.length >= 2
                                  ? colors.yellow
                                  : colors.textDim,
                            padding: '2px 6px',
                            borderRadius: 10,
                          }}
                        >
                          {client.alerts.length}
                        </span>
                      )}
                    </div>
                  </div>
                )
              })}
            </>
          )}
        </div>
      </div>

      {/* Detail panel */}
      {selectedClientId && (
        <div style={{ flex: 1, overflow: 'hidden' }}>
          <ClientDetail clientId={selectedClientId} />
        </div>
      )}
    </div>
  )
}
