export default function SyncingBanner({ message }) {
  if (!message) return null
  return (
    <div style={{
      background: 'var(--indigo-light)',
      borderBottom: '1px solid #c7d2fe',
      padding: '9px 32px',
      fontSize: 13,
      color: 'var(--indigo-dark)',
      display: 'flex',
      alignItems: 'center',
      gap: 8,
    }}>
      <span style={{
        display: 'inline-block',
        width: 7, height: 7,
        borderRadius: '50%',
        background: 'var(--indigo)',
        animation: 'pulse 1.2s ease-in-out infinite',
      }} />
      {message}
    </div>
  )
}
