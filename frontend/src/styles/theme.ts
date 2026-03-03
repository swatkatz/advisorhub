export const colors = {
  bg: '#0A0E17',
  surface: '#111827',
  surfaceHover: '#1a2235',
  surfaceRaised: '#1E293B',
  border: '#1E293B',
  borderLight: '#2a3650',
  text: '#E2E8F0',
  textMuted: '#94A3B8',
  textDim: '#64748B',
  accent: '#22D3EE',
  accentDim: 'rgba(34,211,238,0.1)',
  critical: '#EF4444',
  criticalDim: 'rgba(239,68,68,0.12)',
  urgent: '#F59E0B',
  urgentDim: 'rgba(245,158,11,0.12)',
  advisory: '#818CF8',
  advisoryDim: 'rgba(129,140,248,0.12)',
  info: '#64748B',
  infoDim: 'rgba(100,116,139,0.1)',
  green: '#34D399',
  greenDim: 'rgba(52,211,153,0.12)',
  red: '#EF4444',
  yellow: '#F59E0B',
}

export const severityColors: Record<string, { bg: string; text: string; dot: string }> = {
  CRITICAL: { bg: colors.criticalDim, text: colors.critical, dot: colors.critical },
  URGENT: { bg: colors.urgentDim, text: colors.urgent, dot: colors.urgent },
  ADVISORY: { bg: colors.advisoryDim, text: colors.advisory, dot: colors.advisory },
  INFO: { bg: colors.infoDim, text: colors.info, dot: colors.info },
}

export const healthColors: Record<string, string> = {
  GREEN: colors.green,
  YELLOW: colors.yellow,
  RED: colors.red,
}

export const accountTypeColors: Record<string, { bg: string; text: string }> = {
  RRSP: { bg: 'rgba(129,140,248,0.15)', text: '#A5B4FC' },
  TFSA: { bg: 'rgba(52,211,153,0.15)', text: '#6EE7B7' },
  FHSA: { bg: 'rgba(251,191,36,0.15)', text: '#FCD34D' },
  RESP: { bg: 'rgba(244,114,182,0.15)', text: '#F9A8D4' },
  NON_REG: { bg: 'rgba(148,163,184,0.12)', text: '#94A3B8' },
}

export const actionItemStatusColors: Record<string, string> = {
  PENDING: colors.textDim,
  IN_PROGRESS: colors.accent,
  DONE: colors.green,
  CLOSED: colors.textDim,
}

export const actionItemStatusLabels: Record<string, string> = {
  PENDING: 'Pending',
  IN_PROGRESS: 'In progress',
  DONE: 'Done',
  CLOSED: 'Closed',
}

export const severityOrder: Record<string, number> = {
  CRITICAL: 0,
  URGENT: 1,
  ADVISORY: 2,
  INFO: 3,
}
