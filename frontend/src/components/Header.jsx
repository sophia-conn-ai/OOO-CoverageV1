export default function Header({ onRefresh, refreshing, lastUpdated }) {
  return (
    <header style={{
      background: 'var(--slate-900)',
      color: '#f8fafc',
      padding: '16px 32px',
      display: 'flex',
      alignItems: 'center',
      justifyContent: 'space-between',
      position: 'sticky',
      top: 0,
      zIndex: 10,
      boxShadow: '0 2px 8px rgba(0,0,0,0.25)',
    }}>
      <div>
        <h1 style={{ fontSize: '1.15rem', fontWeight: 700, letterSpacing: '-0.01em' }}>
          OOO Coverage —{' '}
          <span style={{ color: '#818cf8' }}>Sophia Conn</span>
        </h1>
        {lastUpdated && (
          <p style={{ fontSize: 12, color: 'var(--slate-400)', marginTop: 2 }}>
            Updated {new Date(lastUpdated).toLocaleString()}
          </p>
        )}
      </div>

      <div style={{ display: 'flex', alignItems: 'center', gap: 12 }}>
        {refreshing && (
          <span style={{ display: 'flex', alignItems: 'center', gap: 6, fontSize: 12, color: 'var(--slate-400)' }}>
            <PulseDot /> Updating in background…
          </span>
        )}
        <button
          onClick={onRefresh}
          style={{
            padding: '7px 16px',
            background: 'var(--indigo)',
            color: '#fff',
            border: 'none',
            borderRadius: 7,
            cursor: 'pointer',
            fontSize: 13,
            fontWeight: 600,
            fontFamily: 'inherit',
            transition: 'background 0.15s',
          }}
          onMouseOver={e => e.target.style.background = 'var(--indigo-dark)'}
          onMouseOut={e => e.target.style.background = 'var(--indigo)'}
        >
          ↻ Refresh
        </button>
      </div>
    </header>
  )
}

function PulseDot() {
  return (
    <span style={{
      display: 'inline-block',
      width: 7, height: 7,
      borderRadius: '50%',
      background: 'var(--indigo)',
      animation: 'pulse 1.2s ease-in-out infinite',
    }} />
  )
}
