import { useState, useEffect, useRef } from "react";

// ─── Data ───────────────────────────────────────────────────────────────────

const CLIENTS = [
  { id: "c1", name: "Priya Sharma", avatar: "PS", aum: 485000, accounts: ["RRSP", "TFSA", "FHSA"], lastMeeting: "2025-12-14", health: "red", alerts: 3 },
  { id: "c2", name: "Marcus Chen", avatar: "MC", aum: 1250000, accounts: ["RRSP", "TFSA", "RESP", "Non-Reg"], lastMeeting: "2026-01-22", health: "green", alerts: 1 },
  { id: "c3", name: "Swati & Rohan Gupta", avatar: "SG", aum: 720000, accounts: ["RRSP", "TFSA", "FHSA", "RESP"], lastMeeting: "2026-02-10", health: "yellow", alerts: 2 },
  { id: "c4", name: "Elena Vasquez", avatar: "EV", aum: 310000, accounts: ["RRSP", "TFSA"], lastMeeting: "2025-09-05", health: "yellow", alerts: 1 },
  { id: "c5", name: "James & Tanya Williams", avatar: "JW", aum: 2100000, accounts: ["RRSP", "TFSA", "RESP", "Non-Reg"], lastMeeting: "2026-02-25", health: "green", alerts: 0 },
  { id: "c6", name: "Amir Patel", avatar: "AP", aum: 195000, accounts: ["RRSP", "TFSA", "FHSA"], lastMeeting: "2025-11-18", health: "red", alerts: 2 },
  { id: "c7", name: "Sophie Tremblay", avatar: "ST", aum: 890000, accounts: ["RRSP", "TFSA", "Non-Reg"], lastMeeting: "2026-01-30", health: "green", alerts: 1 },
  { id: "c8", name: "David Kim", avatar: "DK", aum: 540000, accounts: ["RRSP", "TFSA", "RESP"], lastMeeting: "2025-08-12", health: "yellow", alerts: 1 },
];

const ALERTS = [
  { id: "a1", clientId: "c1", clientName: "Priya Sharma", severity: "critical", category: "Over-contribution", summary: "Priya has over-contributed $2,300 to her RRSP across Wealthsimple and RBC. At the 1% monthly penalty, this is costing her $23/month. Recommend she withdraw the excess before March to minimize CRA penalties.", time: "2 hours ago", status: "review", draft: "Hi Priya, I noticed your combined RRSP contributions this year have exceeded your limit by $2,300. I'd recommend we withdraw the excess amount promptly to avoid the 1% monthly penalty from CRA. Can we chat this week to sort this out?" },
  { id: "a2", clientId: "c6", clientName: "Amir Patel", severity: "critical", category: "Transfer stuck", summary: "Amir's RRSP transfer from TD ($67,400) has been in 'Documents Submitted' for 18 days. Average for TD is 8-10 days. Likely needs follow-up with TD's transfer department.", time: "4 hours ago", status: "review", draft: "Hi Amir, I wanted to give you an update on your RRSP transfer from TD. It's been longer than expected — I'm going to follow up directly with TD's transfer team to get things moving. I'll keep you posted." },
  { id: "a3", clientId: "c3", clientName: "Swati & Rohan Gupta", severity: "urgent", category: "Deadline approaching", summary: "RRSP deadline is 12 days away (March 3). Swati has $8,200 of contribution room remaining. In your last meeting, she mentioned wanting to maximize RRSP before mat leave ends.", time: "6 hours ago", status: "review", draft: "Hi Swati, Quick reminder — the RRSP deadline is March 3 and you still have $8,200 of room. Given our conversation about maximizing contributions before you return to work, would you like to set up a contribution this week?" },
  { id: "a4", clientId: "c1", clientName: "Priya Sharma", severity: "urgent", category: "CESG matching gap", summary: "Priya contributed $1,800 to her son's RESP this year. An additional $700 would maximize the $500 CESG government grant. Total lifetime RESP contributions: $38,200 of $50,000.", time: "6 hours ago", status: "review", draft: null },
  { id: "a5", clientId: "c2", clientName: "Marcus Chen", severity: "urgent", category: "Age milestone", summary: "Marcus turns 71 in November 2026. His RRSP must be converted to a RRIF by December 31, 2026. Current RRSP balance: $620,000. Recommend discussing drawdown strategy and tax implications in next meeting.", time: "1 day ago", status: "review", draft: null },
  { id: "a6", clientId: "c3", clientName: "Swati & Rohan Gupta", severity: "advisory", category: "Cash uninvested", summary: "Rohan's non-registered account has $45,200 in cash that's been sitting uninvested for 34 days since a large deposit on January 26. Worth discussing investment deployment strategy.", time: "1 day ago", status: "review", draft: null },
  { id: "a7", clientId: "c7", clientName: "Sophie Tremblay", severity: "advisory", category: "Portfolio drift", summary: "Sophie's portfolio has drifted 12% from target allocation — tech sector now at 42% vs 30% target due to recent gains. Rebalancing would reduce concentration risk.", time: "1 day ago", status: "review", draft: null },
  { id: "a8", clientId: "c4", clientName: "Elena Vasquez", severity: "advisory", category: "Engagement", summary: "Last meeting with Elena was 178 days ago (September 5). She has a mortgage renewal coming up in April. Good time to schedule a check-in and discuss rate strategy.", time: "2 days ago", status: "review", draft: null },
  { id: "a9", clientId: "c6", clientName: "Amir Patel", severity: "advisory", category: "Tax-loss harvesting", summary: "Amir holds $12,000 in a Canadian energy ETF with $3,200 unrealized loss. Harvesting before year-end could offset gains from his non-registered account. 30-day superficial loss rule applies.", time: "2 days ago", status: "review", draft: null },
  { id: "a10", clientId: "c8", clientName: "David Kim", severity: "advisory", category: "Engagement", summary: "Last meeting with David was 201 days ago (August 12). He has two RESPs and his oldest child turns 17 next year — RESP strategy discussion needed.", time: "3 days ago", status: "review", draft: null },
  { id: "a11", clientId: "c5", clientName: "James & Tanya Williams", severity: "info", category: "Transfer complete", summary: "James's non-registered account transfer from Scotia ($185,000) completed successfully. Funds are now available for investment.", time: "3 hours ago", status: "auto-sent" },
  { id: "a12", clientId: "c2", clientName: "Marcus Chen", severity: "info", category: "Contribution processed", summary: "Marcus's monthly TFSA contribution of $583.33 was processed successfully. Year-to-date: $1,166.66 of $7,000 room used.", time: "5 hours ago", status: "auto-sent" },
  { id: "a13", clientId: "c7", clientName: "Sophie Tremblay", severity: "info", category: "Dividend received", summary: "Sophie's non-registered account received $1,240 in quarterly dividends from her Canadian dividend ETF holdings.", time: "1 day ago", status: "auto-sent" },
];

const TRANSFERS = [
  { id: "t1", clientName: "Amir Patel", source: "TD", accountType: "RRSP", amount: 67400, stage: "documents", daysInStage: 18, initiated: "2026-02-06", stuck: true },
  { id: "t2", clientName: "James Williams", source: "Scotia", accountType: "Non-Reg", amount: 185000, stage: "complete", daysInStage: 0, initiated: "2026-01-15", stuck: false },
  { id: "t3", clientName: "Priya Sharma", source: "RBC", accountType: "RRSP", amount: 42000, stage: "in-transit", daysInStage: 3, initiated: "2026-02-18", stuck: false },
  { id: "t4", clientName: "David Kim", source: "BMO", accountType: "TFSA", amount: 28500, stage: "review", daysInStage: 5, initiated: "2026-02-20", stuck: false },
  { id: "t5", clientName: "Elena Vasquez", source: "Desjardins", accountType: "RRSP", amount: 55000, stage: "initiated", daysInStage: 2, initiated: "2026-02-27", stuck: false },
  { id: "t6", clientName: "Sophie Tremblay", source: "National Bank", accountType: "Non-Reg", amount: 120000, stage: "in-transit", daysInStage: 6, initiated: "2026-02-15", stuck: false },
];

const TRANSFER_STAGES = [
  { key: "initiated", label: "Initiated" },
  { key: "documents", label: "Documents Submitted" },
  { key: "review", label: "In Review" },
  { key: "in-transit", label: "Assets in Transit" },
  { key: "received", label: "Received at WS" },
  { key: "complete", label: "Invested" },
];

const CLIENT_DETAILS = {
  c1: {
    contributions: [
      { account: "RRSP", room: 31560, contributed: 33860, remaining: -2300, overContributed: true },
      { account: "TFSA", room: 7000, contributed: 4500, remaining: 2500, overContributed: false },
      { account: "FHSA", room: 8000, contributed: 8000, remaining: 0, overContributed: false },
    ],
    externalAccounts: [{ institution: "RBC", type: "RRSP", balance: 45000 }, { institution: "RBC", type: "TFSA", balance: 12000 }],
    goals: [{ name: "Retirement at 60", progress: 62, status: "behind" }, { name: "Son's education (RESP)", progress: 76, status: "on-track" }],
    notes: [
      { date: "2025-12-14", text: "Discussed FHSA eligibility. Priya confirmed first-time homebuyer. Maxed FHSA for 2025." },
      { date: "2025-10-02", text: "Reviewed RESP strategy. On track for CESG matching. Discussed increasing contributions next year." },
    ],
    actionItems: [
      { id: "ai1", text: "Withdraw $2,300 RRSP over-contribution", status: "urgent", due: "2026-03-10" },
      { id: "ai2", text: "Top up RESP by $700 for CESG match", status: "pending", due: "2026-12-31" },
      { id: "ai3", text: "Review RBC account consolidation", status: "in-progress", due: "2026-04-01" },
    ],
  },
  c3: {
    contributions: [
      { account: "RRSP", room: 32490, contributed: 24290, remaining: 8200, overContributed: false },
      { account: "TFSA", room: 7000, contributed: 7000, remaining: 0, overContributed: false },
      { account: "FHSA", room: 8000, contributed: 5000, remaining: 3000, overContributed: false },
      { account: "RESP", room: 2500, contributed: 2500, remaining: 0, overContributed: false },
    ],
    externalAccounts: [{ institution: "Desjardins", type: "RRSP", balance: 18000 }],
    goals: [{ name: "Retirement at 58", progress: 71, status: "on-track" }, { name: "Daughter's education", progress: 15, status: "on-track" }, { name: "First home (FHSA)", progress: 44, status: "on-track" }],
    notes: [
      { date: "2026-02-10", text: "Swati on mat leave until June. Wants to maximize RRSP before returning. Discussed FHSA timeline — planning to buy in 2-3 years." },
      { date: "2025-11-20", text: "Rohan received large bonus. Deposited $45K to non-reg. Will discuss investment strategy at next meeting." },
    ],
    actionItems: [
      { id: "ai4", text: "Contribute remaining $8,200 to RRSP before March 3 deadline", status: "urgent", due: "2026-03-03" },
      { id: "ai5", text: "Deploy $45K cash in non-registered account", status: "pending", due: "2026-03-15" },
      { id: "ai6", text: "Review FHSA contribution room for 2026", status: "pending", due: "2026-04-01" },
    ],
  },
};

// ─── Styles ─────────────────────────────────────────────────────────────────

const colors = {
  bg: "#0A0E17",
  surface: "#111827",
  surfaceHover: "#1a2235",
  surfaceRaised: "#1E293B",
  border: "#1E293B",
  borderLight: "#2a3650",
  text: "#E2E8F0",
  textMuted: "#94A3B8",
  textDim: "#64748B",
  accent: "#22D3EE",
  accentDim: "rgba(34,211,238,0.1)",
  critical: "#EF4444",
  criticalDim: "rgba(239,68,68,0.12)",
  urgent: "#F59E0B",
  urgentDim: "rgba(245,158,11,0.12)",
  advisory: "#818CF8",
  advisoryDim: "rgba(129,140,248,0.12)",
  info: "#64748B",
  infoDim: "rgba(100,116,139,0.1)",
  green: "#34D399",
  greenDim: "rgba(52,211,153,0.12)",
  red: "#EF4444",
  yellow: "#F59E0B",
};

const severityColors = {
  critical: { bg: colors.criticalDim, text: colors.critical, dot: colors.critical },
  urgent: { bg: colors.urgentDim, text: colors.urgent, dot: colors.urgent },
  advisory: { bg: colors.advisoryDim, text: colors.advisory, dot: colors.advisory },
  info: { bg: colors.infoDim, text: colors.info, dot: colors.info },
};

const healthColors = { green: colors.green, yellow: colors.yellow, red: colors.red };

// ─── Components ─────────────────────────────────────────────────────────────

function Badge({ children, color, bgColor }) {
  return (
    <span style={{ fontSize: 11, fontWeight: 600, letterSpacing: "0.04em", textTransform: "uppercase", color, background: bgColor, padding: "3px 8px", borderRadius: 4 }}>
      {children}
    </span>
  );
}

function SeverityDot({ severity }) {
  const c = severityColors[severity];
  return (
    <span style={{ width: 8, height: 8, borderRadius: "50%", background: c.dot, boxShadow: `0 0 6px ${c.dot}`, flexShrink: 0, marginTop: 6 }} />
  );
}

function IconButton({ children, onClick, title, variant = "default" }) {
  const [hovered, setHovered] = useState(false);
  const styles = {
    default: { bg: hovered ? colors.surfaceRaised : "transparent", border: colors.borderLight, color: colors.textMuted },
    send: { bg: hovered ? "rgba(34,211,238,0.15)" : colors.accentDim, border: "rgba(34,211,238,0.3)", color: colors.accent },
    dismiss: { bg: hovered ? "rgba(239,68,68,0.12)" : "transparent", border: colors.borderLight, color: colors.textMuted },
  };
  const s = styles[variant];
  return (
    <button
      onClick={onClick}
      title={title}
      onMouseEnter={() => setHovered(true)}
      onMouseLeave={() => setHovered(false)}
      style={{ display: "flex", alignItems: "center", justifyContent: "center", width: 32, height: 32, borderRadius: 6, background: s.bg, border: `1px solid ${s.border}`, color: s.color, cursor: "pointer", transition: "all 0.15s" }}
    >
      {children}
    </button>
  );
}

function Avatar({ initials, size = 36 }) {
  const hue = initials.charCodeAt(0) * 7 + initials.charCodeAt(1) * 13;
  const bg = `hsl(${hue % 360}, 40%, 25%)`;
  const fg = `hsl(${hue % 360}, 60%, 75%)`;
  return (
    <div style={{ width: size, height: size, borderRadius: "50%", background: bg, color: fg, display: "flex", alignItems: "center", justifyContent: "center", fontSize: size * 0.36, fontWeight: 600, letterSpacing: "0.02em", flexShrink: 0 }}>
      {initials}
    </div>
  );
}

function AccountTag({ type }) {
  const tagColors = {
    RRSP: { bg: "rgba(129,140,248,0.15)", text: "#A5B4FC" },
    TFSA: { bg: "rgba(52,211,153,0.15)", text: "#6EE7B7" },
    FHSA: { bg: "rgba(251,191,36,0.15)", text: "#FCD34D" },
    RESP: { bg: "rgba(244,114,182,0.15)", text: "#F9A8D4" },
    "Non-Reg": { bg: "rgba(148,163,184,0.12)", text: "#94A3B8" },
  };
  const c = tagColors[type] || tagColors["Non-Reg"];
  return <span style={{ fontSize: 10, fontWeight: 600, padding: "2px 6px", borderRadius: 3, background: c.bg, color: c.text, letterSpacing: "0.03em" }}>{type}</span>;
}

// ─── Alert Feed ─────────────────────────────────────────────────────────────

function AlertCard({ alert, onSend, onDismiss, onExpand }) {
  const [expanded, setExpanded] = useState(false);
  const sc = severityColors[alert.severity];
  const isSent = alert.status === "auto-sent" || alert.status === "sent";

  return (
    <div
      style={{
        background: colors.surface,
        border: `1px solid ${colors.border}`,
        borderLeft: `3px solid ${sc.dot}`,
        borderRadius: 8,
        padding: "14px 16px",
        marginBottom: 8,
        opacity: isSent ? 0.65 : 1,
        transition: "all 0.2s",
      }}
    >
      <div style={{ display: "flex", gap: 12 }}>
        <SeverityDot severity={alert.severity} />
        <div style={{ flex: 1, minWidth: 0 }}>
          <div style={{ display: "flex", alignItems: "center", gap: 8, marginBottom: 4, flexWrap: "wrap" }}>
            <span style={{ fontSize: 13, fontWeight: 600, color: colors.text }}>{alert.clientName}</span>
            <Badge color={sc.text} bgColor={sc.bg}>{alert.category}</Badge>
            {isSent && <Badge color={colors.green} bgColor={colors.greenDim}>Sent ✓</Badge>}
            <span style={{ fontSize: 11, color: colors.textDim, marginLeft: "auto", flexShrink: 0 }}>{alert.time}</span>
          </div>
          <p style={{ fontSize: 13, lineHeight: 1.55, color: colors.textMuted, margin: 0 }}>{alert.summary}</p>

          {alert.draft && !isSent && (
            <button
              onClick={() => setExpanded(!expanded)}
              style={{ marginTop: 8, fontSize: 12, color: colors.accent, background: "none", border: "none", cursor: "pointer", padding: 0, fontFamily: "inherit" }}
            >
              {expanded ? "Hide draft ▴" : "Preview draft ▾"}
            </button>
          )}

          {expanded && alert.draft && (
            <div style={{ marginTop: 8, padding: "10px 12px", background: colors.surfaceRaised, borderRadius: 6, border: `1px solid ${colors.borderLight}`, fontSize: 12.5, lineHeight: 1.6, color: colors.textMuted, fontStyle: "italic" }}>
              {alert.draft}
            </div>
          )}
        </div>

        {!isSent && (
          <div style={{ display: "flex", flexDirection: "column", gap: 4, flexShrink: 0 }}>
            <IconButton variant="send" title="Send to client" onClick={() => onSend?.(alert.id)}>
              <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.5" strokeLinecap="round" strokeLinejoin="round"><path d="M22 2L11 13"/><path d="M22 2L15 22L11 13L2 9L22 2Z"/></svg>
            </IconButton>
            <IconButton variant="dismiss" title="Snooze" onClick={() => onDismiss?.(alert.id)}>
              <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><circle cx="12" cy="12" r="10"/><path d="M12 6v6l4 2"/></svg>
            </IconButton>
          </div>
        )}
      </div>
    </div>
  );
}

function AlertFeed() {
  const [filter, setFilter] = useState("all");
  const [alerts, setAlerts] = useState(ALERTS);
  const [sweepRunning, setSweepRunning] = useState(false);
  const [sweepCount, setSweepCount] = useState(0);

  const handleSend = (id) => {
    setAlerts((prev) => prev.map((a) => (a.id === id ? { ...a, status: "sent" } : a)));
  };

  const handleDismiss = (id) => {
    setAlerts((prev) => prev.filter((a) => a.id !== id));
  };

  const runSweep = () => {
    setSweepRunning(true);
    setSweepCount(0);
    const newAlerts = [
      { id: "sweep1", clientId: "c8", clientName: "David Kim", severity: "urgent", category: "RESP strategy", summary: "David's oldest child turns 17 next year. RESP contributions after age 17 don't qualify for CESG. Current balance: $32,400 of $50,000 lifetime limit. Recommend accelerating contributions this year.", time: "Just now", status: "review", draft: null },
      { id: "sweep2", clientId: "c4", clientName: "Elena Vasquez", severity: "advisory", category: "Mortgage renewal", summary: "Elena's mortgage renews in April 2026. Current rate: 4.2% fixed. Market rates have dropped. Scheduling a meeting to discuss refinancing options could save her $340/month.", time: "Just now", status: "review", draft: null },
    ];
    let i = 0;
    const interval = setInterval(() => {
      if (i < newAlerts.length) {
        setAlerts((prev) => [newAlerts[i], ...prev]);
        setSweepCount((c) => c + 1);
        i++;
      } else {
        clearInterval(interval);
        setSweepRunning(false);
      }
    }, 800);
  };

  const filtered = filter === "all" ? alerts : filter === "attention" ? alerts.filter((a) => a.status === "review") : alerts.filter((a) => a.severity === filter);
  const attentionCount = alerts.filter((a) => a.status === "review").length;

  return (
    <div style={{ height: "100%", display: "flex", flexDirection: "column" }}>
      {/* Header */}
      <div style={{ padding: "20px 20px 0" }}>
        <div style={{ display: "flex", alignItems: "center", justifyContent: "space-between", marginBottom: 16 }}>
          <div>
            <h2 style={{ fontSize: 18, fontWeight: 700, color: colors.text, margin: 0 }}>Alert Feed</h2>
            <p style={{ fontSize: 12, color: colors.textDim, margin: "4px 0 0" }}>{attentionCount} alerts need your attention</p>
          </div>
          <button
            onClick={runSweep}
            disabled={sweepRunning}
            style={{
              display: "flex", alignItems: "center", gap: 6, padding: "8px 14px", borderRadius: 6,
              background: sweepRunning ? colors.surfaceRaised : `linear-gradient(135deg, rgba(34,211,238,0.15), rgba(129,140,248,0.15))`,
              border: `1px solid ${sweepRunning ? colors.borderLight : "rgba(34,211,238,0.3)"}`,
              color: sweepRunning ? colors.textDim : colors.accent, fontSize: 12, fontWeight: 600, cursor: sweepRunning ? "default" : "pointer", fontFamily: "inherit",
            }}
          >
            {sweepRunning ? (
              <><span style={{ display: "inline-block", animation: "spin 1s linear infinite", fontSize: 13 }}>⟳</span> Scanning... ({sweepCount} found)</>
            ) : (
              <><svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.5"><path d="M21 12a9 9 0 11-6.22-8.56"/><path d="M21 3v9h-9"/></svg> Run morning sweep</>
            )}
          </button>
        </div>

        {/* Filters */}
        <div style={{ display: "flex", gap: 6, marginBottom: 14 }}>
          {[
            { key: "all", label: "All" },
            { key: "attention", label: "Needs attention" },
            { key: "critical", label: "Critical" },
            { key: "urgent", label: "Urgent" },
            { key: "advisory", label: "Advisory" },
          ].map((f) => (
            <button
              key={f.key}
              onClick={() => setFilter(f.key)}
              style={{
                padding: "5px 10px", borderRadius: 5, fontSize: 11, fontWeight: 500, cursor: "pointer", fontFamily: "inherit", transition: "all 0.15s",
                background: filter === f.key ? colors.surfaceRaised : "transparent",
                border: `1px solid ${filter === f.key ? colors.borderLight : "transparent"}`,
                color: filter === f.key ? colors.text : colors.textDim,
              }}
            >
              {f.label}
            </button>
          ))}
        </div>
      </div>

      {/* Alert list */}
      <div style={{ flex: 1, overflow: "auto", padding: "0 20px 20px" }}>
        {filtered.map((alert) => (
          <AlertCard key={alert.id} alert={alert} onSend={handleSend} onDismiss={handleDismiss} />
        ))}
      </div>
    </div>
  );
}

// ─── Transfer Tracking ──────────────────────────────────────────────────────

function TransferCard({ transfer }) {
  return (
    <div style={{
      background: colors.surface,
      border: `1px solid ${transfer.stuck ? "rgba(239,68,68,0.4)" : colors.border}`,
      borderRadius: 8, padding: "10px 12px", marginBottom: 6,
      boxShadow: transfer.stuck ? "0 0 12px rgba(239,68,68,0.08)" : "none",
    }}>
      <div style={{ display: "flex", alignItems: "center", justifyContent: "space-between", marginBottom: 6 }}>
        <span style={{ fontSize: 13, fontWeight: 600, color: colors.text }}>{transfer.clientName}</span>
        <AccountTag type={transfer.accountType} />
      </div>
      <div style={{ display: "flex", alignItems: "center", justifyContent: "space-between", marginBottom: 4 }}>
        <span style={{ fontSize: 12, color: colors.textMuted }}>From {transfer.source}</span>
        <span style={{ fontSize: 13, fontWeight: 600, color: colors.text }}>${transfer.amount.toLocaleString()}</span>
      </div>
      <div style={{ display: "flex", alignItems: "center", justifyContent: "space-between" }}>
        <span style={{ fontSize: 11, color: colors.textDim }}>Initiated {transfer.initiated}</span>
        {transfer.stuck ? (
          <span style={{ fontSize: 11, fontWeight: 600, color: colors.red }}>⚠ Stuck {transfer.daysInStage}d</span>
        ) : (
          <span style={{ fontSize: 11, color: colors.textDim }}>{transfer.daysInStage}d in stage</span>
        )}
      </div>
    </div>
  );
}

function TransferTracking() {
  return (
    <div style={{ height: "100%", display: "flex", flexDirection: "column" }}>
      <div style={{ padding: "20px 20px 0" }}>
        <h2 style={{ fontSize: 18, fontWeight: 700, color: colors.text, margin: 0 }}>Transfer Tracking</h2>
        <p style={{ fontSize: 12, color: colors.textDim, margin: "4px 0 16px" }}>{TRANSFERS.length} active transfers across clients</p>
      </div>
      <div style={{ flex: 1, overflow: "auto", padding: "0 20px 20px" }}>
        <div style={{ display: "flex", gap: 12, minWidth: "max-content", height: "100%" }}>
          {TRANSFER_STAGES.map((stage) => {
            const transfers = TRANSFERS.filter((t) => t.stage === stage.key);
            return (
              <div key={stage.key} style={{ width: 220, flexShrink: 0 }}>
                <div style={{ display: "flex", alignItems: "center", gap: 6, marginBottom: 10, padding: "0 2px" }}>
                  <span style={{ fontSize: 12, fontWeight: 600, color: colors.textMuted, letterSpacing: "0.02em" }}>{stage.label}</span>
                  {transfers.length > 0 && (
                    <span style={{ fontSize: 10, fontWeight: 700, color: colors.textDim, background: colors.surfaceRaised, padding: "2px 6px", borderRadius: 10 }}>
                      {transfers.length}
                    </span>
                  )}
                </div>
                <div style={{ minHeight: 80, background: "rgba(255,255,255,0.015)", borderRadius: 8, padding: 6 }}>
                  {transfers.map((t) => <TransferCard key={t.id} transfer={t} />)}
                  {transfers.length === 0 && (
                    <div style={{ padding: 20, textAlign: "center", fontSize: 11, color: colors.textDim }}>—</div>
                  )}
                </div>
              </div>
            );
          })}
        </div>
      </div>
    </div>
  );
}

// ─── Client Book ────────────────────────────────────────────────────────────

function ContributionBar({ account, room, contributed, remaining, overContributed }) {
  const pct = Math.min((contributed / room) * 100, 100);
  const overPct = overContributed ? ((contributed - room) / room) * 100 : 0;
  return (
    <div style={{ marginBottom: 10 }}>
      <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center", marginBottom: 4 }}>
        <AccountTag type={account} />
        <span style={{ fontSize: 11, color: overContributed ? colors.red : remaining === 0 ? colors.green : colors.textMuted, fontWeight: 600 }}>
          {overContributed ? `Over by $${Math.abs(remaining).toLocaleString()}` : remaining === 0 ? "Maxed ✓" : `$${remaining.toLocaleString()} room`}
        </span>
      </div>
      <div style={{ height: 6, background: colors.surfaceRaised, borderRadius: 3, overflow: "hidden", position: "relative" }}>
        <div style={{
          position: "absolute", left: 0, top: 0, height: "100%", borderRadius: 3, transition: "width 0.5s ease",
          width: `${Math.min(pct, 100)}%`,
          background: overContributed ? colors.red : pct >= 100 ? colors.green : colors.accent,
        }} />
      </div>
      <div style={{ display: "flex", justifyContent: "space-between", marginTop: 3 }}>
        <span style={{ fontSize: 10, color: colors.textDim }}>${contributed.toLocaleString()} contributed</span>
        <span style={{ fontSize: 10, color: colors.textDim }}>${room.toLocaleString()} limit</span>
      </div>
    </div>
  );
}

function GoalBar({ name, progress, status }) {
  const goalColor = status === "behind" ? colors.yellow : colors.green;
  return (
    <div style={{ marginBottom: 10 }}>
      <div style={{ display: "flex", justifyContent: "space-between", marginBottom: 4 }}>
        <span style={{ fontSize: 12, color: colors.text }}>{name}</span>
        <span style={{ fontSize: 11, fontWeight: 600, color: goalColor }}>{progress}%</span>
      </div>
      <div style={{ height: 4, background: colors.surfaceRaised, borderRadius: 2 }}>
        <div style={{ height: "100%", width: `${progress}%`, background: goalColor, borderRadius: 2, transition: "width 0.5s ease" }} />
      </div>
    </div>
  );
}

function ClientDetail({ client }) {
  const details = CLIENT_DETAILS[client.id];
  if (!details) {
    return (
      <div style={{ padding: 30, textAlign: "center", color: colors.textDim, fontSize: 13 }}>
        Client detail view — data available for Priya Sharma and Swati & Rohan Gupta
      </div>
    );
  }

  const statusColors = { urgent: colors.red, "in-progress": colors.accent, pending: colors.textDim, done: colors.green };
  const statusLabels = { urgent: "Urgent", "in-progress": "In progress", pending: "Pending", done: "Done" };

  return (
    <div style={{ padding: 20, overflow: "auto", height: "100%" }}>
      {/* Header */}
      <div style={{ display: "flex", alignItems: "center", gap: 12, marginBottom: 20 }}>
        <Avatar initials={client.avatar} size={44} />
        <div>
          <h3 style={{ fontSize: 16, fontWeight: 700, color: colors.text, margin: 0 }}>{client.name}</h3>
          <p style={{ fontSize: 12, color: colors.textDim, margin: "2px 0 0" }}>AUM: ${client.aum.toLocaleString()} · Last meeting: {client.lastMeeting}</p>
        </div>
      </div>

      {/* Contribution Summary */}
      <div style={{ marginBottom: 20 }}>
        <h4 style={{ fontSize: 13, fontWeight: 600, color: colors.textMuted, letterSpacing: "0.04em", textTransform: "uppercase", margin: "0 0 12px", borderBottom: `1px solid ${colors.border}`, paddingBottom: 8 }}>
          Contribution Room — 2026
        </h4>
        {details.contributions.map((c) => <ContributionBar key={c.account} {...c} />)}
      </div>

      {/* External Accounts */}
      {details.externalAccounts.length > 0 && (
        <div style={{ marginBottom: 20 }}>
          <h4 style={{ fontSize: 13, fontWeight: 600, color: colors.textMuted, letterSpacing: "0.04em", textTransform: "uppercase", margin: "0 0 10px", borderBottom: `1px solid ${colors.border}`, paddingBottom: 8 }}>
            External Accounts
          </h4>
          {details.externalAccounts.map((ea, i) => (
            <div key={i} style={{ display: "flex", justifyContent: "space-between", alignItems: "center", padding: "6px 0", borderBottom: i < details.externalAccounts.length - 1 ? `1px solid ${colors.border}` : "none" }}>
              <div style={{ display: "flex", alignItems: "center", gap: 8 }}>
                <span style={{ fontSize: 12, color: colors.text }}>{ea.institution}</span>
                <AccountTag type={ea.type} />
              </div>
              <span style={{ fontSize: 13, fontWeight: 600, color: colors.text }}>${ea.balance.toLocaleString()}</span>
            </div>
          ))}
        </div>
      )}

      {/* Goals */}
      <div style={{ marginBottom: 20 }}>
        <h4 style={{ fontSize: 13, fontWeight: 600, color: colors.textMuted, letterSpacing: "0.04em", textTransform: "uppercase", margin: "0 0 12px", borderBottom: `1px solid ${colors.border}`, paddingBottom: 8 }}>
          Goals
        </h4>
        {details.goals.map((g) => <GoalBar key={g.name} {...g} />)}
      </div>

      {/* Action Items */}
      <div style={{ marginBottom: 20 }}>
        <h4 style={{ fontSize: 13, fontWeight: 600, color: colors.textMuted, letterSpacing: "0.04em", textTransform: "uppercase", margin: "0 0 10px", borderBottom: `1px solid ${colors.border}`, paddingBottom: 8 }}>
          Action Items
        </h4>
        {details.actionItems.map((ai) => (
          <div key={ai.id} style={{ display: "flex", alignItems: "flex-start", gap: 8, padding: "8px 0", borderBottom: `1px solid ${colors.border}` }}>
            <div style={{ width: 6, height: 6, borderRadius: "50%", background: statusColors[ai.status], marginTop: 5, flexShrink: 0 }} />
            <div style={{ flex: 1 }}>
              <span style={{ fontSize: 12.5, color: colors.text }}>{ai.text}</span>
              <div style={{ display: "flex", gap: 8, marginTop: 3 }}>
                <span style={{ fontSize: 10, color: statusColors[ai.status], fontWeight: 600 }}>{statusLabels[ai.status]}</span>
                <span style={{ fontSize: 10, color: colors.textDim }}>Due: {ai.due}</span>
              </div>
            </div>
          </div>
        ))}
      </div>

      {/* Notes */}
      <div>
        <h4 style={{ fontSize: 13, fontWeight: 600, color: colors.textMuted, letterSpacing: "0.04em", textTransform: "uppercase", margin: "0 0 10px", borderBottom: `1px solid ${colors.border}`, paddingBottom: 8 }}>
          Advisor Notes
        </h4>
        {details.notes.map((n, i) => (
          <div key={i} style={{ padding: "8px 0", borderBottom: i < details.notes.length - 1 ? `1px solid ${colors.border}` : "none" }}>
            <span style={{ fontSize: 10, fontWeight: 600, color: colors.textDim }}>{n.date}</span>
            <p style={{ fontSize: 12.5, color: colors.textMuted, margin: "4px 0 0", lineHeight: 1.5 }}>{n.text}</p>
          </div>
        ))}
        <button style={{
          marginTop: 10, width: "100%", padding: "8px", borderRadius: 6, fontSize: 12, fontWeight: 500,
          background: "transparent", border: `1px dashed ${colors.borderLight}`, color: colors.textDim, cursor: "pointer", fontFamily: "inherit",
        }}>
          + Add note
        </button>
      </div>
    </div>
  );
}

function ClientBook() {
  const [selectedClient, setSelectedClient] = useState(null);
  const [searchQuery, setSearchQuery] = useState("");
  
  const filtered = CLIENTS.filter((c) => c.name.toLowerCase().includes(searchQuery.toLowerCase()));

  return (
    <div style={{ height: "100%", display: "flex" }}>
      {/* Client list */}
      <div style={{ width: selectedClient ? 340 : "100%", borderRight: selectedClient ? `1px solid ${colors.border}` : "none", display: "flex", flexDirection: "column", transition: "width 0.2s" }}>
        <div style={{ padding: "20px 20px 0" }}>
          <h2 style={{ fontSize: 18, fontWeight: 700, color: colors.text, margin: "0 0 4px" }}>Client Book</h2>
          <p style={{ fontSize: 12, color: colors.textDim, margin: "0 0 14px" }}>{CLIENTS.length} clients · ${(CLIENTS.reduce((s, c) => s + c.aum, 0) / 1000000).toFixed(1)}M AUM</p>
          <input
            type="text"
            placeholder="Search clients..."
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            style={{
              width: "100%", padding: "8px 12px", borderRadius: 6, fontSize: 12, fontFamily: "inherit",
              background: colors.surfaceRaised, border: `1px solid ${colors.border}`, color: colors.text,
              outline: "none", marginBottom: 12, boxSizing: "border-box",
            }}
          />
        </div>
        <div style={{ flex: 1, overflow: "auto", padding: "0 20px 20px" }}>
          {/* Table header */}
          <div style={{ display: "grid", gridTemplateColumns: "1fr 100px 90px 80px 50px", gap: 8, padding: "6px 0", borderBottom: `1px solid ${colors.border}`, marginBottom: 4 }}>
            {["Client", "Accounts", "AUM", "Last met", ""].map((h) => (
              <span key={h} style={{ fontSize: 10, fontWeight: 600, color: colors.textDim, letterSpacing: "0.05em", textTransform: "uppercase" }}>{h}</span>
            ))}
          </div>
          {filtered.map((client) => (
            <div
              key={client.id}
              onClick={() => setSelectedClient(selectedClient?.id === client.id ? null : client)}
              style={{
                display: "grid", gridTemplateColumns: "1fr 100px 90px 80px 50px", gap: 8, padding: "10px 0",
                borderBottom: `1px solid ${colors.border}`, cursor: "pointer", alignItems: "center",
                background: selectedClient?.id === client.id ? colors.surfaceRaised : "transparent",
                borderRadius: selectedClient?.id === client.id ? 6 : 0,
                marginLeft: selectedClient?.id === client.id ? -8 : 0,
                marginRight: selectedClient?.id === client.id ? -8 : 0,
                paddingLeft: selectedClient?.id === client.id ? 8 : 0,
                paddingRight: selectedClient?.id === client.id ? 8 : 0,
              }}
            >
              <div style={{ display: "flex", alignItems: "center", gap: 8 }}>
                <Avatar initials={client.avatar} size={28} />
                <span style={{ fontSize: 13, fontWeight: 500, color: colors.text }}>{client.name}</span>
                <span style={{ width: 7, height: 7, borderRadius: "50%", background: healthColors[client.health] }} />
              </div>
              <div style={{ display: "flex", gap: 3, flexWrap: "wrap" }}>
                {client.accounts.map((a) => <AccountTag key={a} type={a} />)}
              </div>
              <span style={{ fontSize: 12, color: colors.text, fontWeight: 500 }}>${(client.aum / 1000).toFixed(0)}K</span>
              <span style={{ fontSize: 11, color: colors.textDim }}>{client.lastMeeting}</span>
              <div>
                {client.alerts > 0 && (
                  <span style={{ fontSize: 10, fontWeight: 700, color: colors.bg, background: client.alerts >= 3 ? colors.red : client.alerts >= 2 ? colors.yellow : colors.textDim, padding: "2px 6px", borderRadius: 10 }}>
                    {client.alerts}
                  </span>
                )}
              </div>
            </div>
          ))}
        </div>
      </div>

      {/* Detail panel */}
      {selectedClient && (
        <div style={{ flex: 1, overflow: "hidden" }}>
          <ClientDetail client={selectedClient} />
        </div>
      )}
    </div>
  );
}

// ─── Main Dashboard ─────────────────────────────────────────────────────────

const NAV_ITEMS = [
  { key: "alerts", label: "Alert Feed", icon: (
    <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><path d="M18 8A6 6 0 006 8c0 7-3 9-3 9h18s-3-2-3-9"/><path d="M13.73 21a2 2 0 01-3.46 0"/></svg>
  )},
  { key: "transfers", label: "Transfers", icon: (
    <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><path d="M17 1l4 4-4 4"/><path d="M3 11V9a4 4 0 014-4h14"/><path d="M7 23l-4-4 4-4"/><path d="M21 13v2a4 4 0 01-4 4H3"/></svg>
  )},
  { key: "clients", label: "Client Book", icon: (
    <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><path d="M17 21v-2a4 4 0 00-4-4H5a4 4 0 00-4 4v2"/><circle cx="9" cy="7" r="4"/><path d="M23 21v-2a4 4 0 00-3-3.87"/><path d="M16 3.13a4 4 0 010 7.75"/></svg>
  )},
];

export default function AdvisorDashboard() {
  const [activeSection, setActiveSection] = useState("alerts");
  const alertCount = ALERTS.filter((a) => a.status === "review").length;

  return (
    <div style={{ display: "flex", height: "100vh", width: "100vw", background: colors.bg, fontFamily: "'DM Sans', 'Segoe UI', system-ui, sans-serif", color: colors.text, overflow: "hidden" }}>
      <style>{`
        @import url('https://fonts.googleapis.com/css2?family=DM+Sans:ital,opsz,wght@0,9..40,300;0,9..40,400;0,9..40,500;0,9..40,600;0,9..40,700&display=swap');
        * { box-sizing: border-box; margin: 0; padding: 0; }
        ::-webkit-scrollbar { width: 6px; }
        ::-webkit-scrollbar-track { background: transparent; }
        ::-webkit-scrollbar-thumb { background: ${colors.borderLight}; border-radius: 3px; }
        ::-webkit-scrollbar-thumb:hover { background: ${colors.textDim}; }
        @keyframes spin { from { transform: rotate(0deg); } to { transform: rotate(360deg); } }
      `}</style>

      {/* Sidebar */}
      <div style={{ width: 220, background: colors.surface, borderRight: `1px solid ${colors.border}`, display: "flex", flexDirection: "column", flexShrink: 0 }}>
        {/* Logo */}
        <div style={{ padding: "20px 16px", borderBottom: `1px solid ${colors.border}` }}>
          <div style={{ display: "flex", alignItems: "center", gap: 10 }}>
            <div style={{ width: 32, height: 32, borderRadius: 8, background: `linear-gradient(135deg, ${colors.accent}, #818CF8)`, display: "flex", alignItems: "center", justifyContent: "center" }}>
              <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="white" strokeWidth="2.5" strokeLinecap="round" strokeLinejoin="round"><path d="M12 2L2 7l10 5 10-5-10-5z"/><path d="M2 17l10 5 10-5"/><path d="M2 12l10 5 10-5"/></svg>
            </div>
            <div>
              <div style={{ fontSize: 14, fontWeight: 700, color: colors.text, letterSpacing: "-0.01em" }}>AdvisorHub</div>
              <div style={{ fontSize: 10, color: colors.textDim }}>by Wealthsimple</div>
            </div>
          </div>
        </div>

        {/* Advisor info */}
        <div style={{ padding: "14px 16px", borderBottom: `1px solid ${colors.border}` }}>
          <div style={{ display: "flex", alignItems: "center", gap: 10 }}>
            <Avatar initials="SK" size={32} />
            <div>
              <div style={{ fontSize: 13, fontWeight: 600, color: colors.text }}>Shruti K.</div>
              <div style={{ fontSize: 10, color: colors.textDim }}>Sr. Financial Advisor</div>
            </div>
          </div>
        </div>

        {/* Nav */}
        <nav style={{ padding: "12px 8px", flex: 1 }}>
          {NAV_ITEMS.map((item) => {
            const isActive = activeSection === item.key;
            return (
              <button
                key={item.key}
                onClick={() => setActiveSection(item.key)}
                style={{
                  display: "flex", alignItems: "center", gap: 10, width: "100%", padding: "10px 12px",
                  borderRadius: 8, border: "none", cursor: "pointer", marginBottom: 2, fontFamily: "inherit",
                  background: isActive ? colors.surfaceRaised : "transparent",
                  color: isActive ? colors.text : colors.textMuted,
                  transition: "all 0.15s",
                  position: "relative",
                }}
              >
                {isActive && <div style={{ position: "absolute", left: 0, top: "50%", transform: "translateY(-50%)", width: 3, height: 20, borderRadius: 2, background: colors.accent }} />}
                {item.icon}
                <span style={{ fontSize: 13, fontWeight: isActive ? 600 : 400 }}>{item.label}</span>
                {item.key === "alerts" && alertCount > 0 && (
                  <span style={{ marginLeft: "auto", fontSize: 10, fontWeight: 700, color: "#fff", background: colors.critical, padding: "2px 6px", borderRadius: 10, minWidth: 18, textAlign: "center" }}>
                    {alertCount}
                  </span>
                )}
              </button>
            );
          })}
        </nav>

        {/* Sweep status */}
        <div style={{ padding: "12px 16px", borderTop: `1px solid ${colors.border}` }}>
          <div style={{ fontSize: 10, color: colors.textDim }}>Last sweep</div>
          <div style={{ fontSize: 12, color: colors.textMuted, marginTop: 2 }}>Today 6:02 AM · 14 alerts</div>
          <div style={{ marginTop: 8, display: "flex", gap: 12 }}>
            <div>
              <div style={{ fontSize: 18, fontWeight: 700, color: colors.text }}>8</div>
              <div style={{ fontSize: 9, color: colors.textDim, textTransform: "uppercase", letterSpacing: "0.05em" }}>Clients</div>
            </div>
            <div>
              <div style={{ fontSize: 18, fontWeight: 700, color: colors.text }}>$6.5M</div>
              <div style={{ fontSize: 9, color: colors.textDim, textTransform: "uppercase", letterSpacing: "0.05em" }}>Total AUM</div>
            </div>
          </div>
        </div>
      </div>

      {/* Main content */}
      <div style={{ flex: 1, overflow: "hidden" }}>
        {activeSection === "alerts" && <AlertFeed />}
        {activeSection === "transfers" && <TransferTracking />}
        {activeSection === "clients" && <ClientBook />}
      </div>
    </div>
  );
}
