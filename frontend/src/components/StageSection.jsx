import { useState } from 'react'
import CandidateRow from './CandidateRow'

export default function StageSection({ stage, candidates, assignments, teamMembers, onUpdate }) {
  const [collapsed, setCollapsed] = useState(false)

  const supportCount = candidates.filter(c => assignments[c.id]?.supportNeeded).length

  return (
    <div style={{
      background: '#fff',
      border: '1px solid var(--slate-200)',
      borderRadius: 12,
      marginBottom: 14,
      overflow: 'hidden',
      boxShadow: '0 1px 3px rgba(0,0,0,0.05)',
    }}>
      {/* Stage header */}
      <div
        onClick={() => setCollapsed(c => !c)}
        style={{
          display: 'flex',
          alignItems: 'center',
          gap: 10,
          padding: '12px 20px',
          cursor: 'pointer',
          userSelect: 'none',
          background: 'var(--slate-50)',
          borderBottom: collapsed ? 'none' : '1px solid var(--slate-200)',
          transition: 'background 0.1s',
        }}
        onMouseOver={e => e.currentTarget.style.background = 'var(--slate-100)'}
        onMouseOut={e => e.currentTarget.style.background = 'var(--slate-50)'}
      >
        <span style={{ width: 9, height: 9, borderRadius: '50%', background: 'var(--indigo)', flexShrink: 0 }} />
        <span style={{ fontWeight: 700, fontSize: 14, color: 'var(--slate-900)' }}>{stage}</span>
        <span style={{
          background: 'var(--indigo-light)',
          color: 'var(--indigo-dark)',
          fontSize: 12,
          fontWeight: 700,
          padding: '2px 9px',
          borderRadius: 20,
        }}>
          {candidates.length}
        </span>
        {supportCount > 0 && (
          <span style={{
            background: '#fff7ed',
            color: 'var(--orange)',
            fontSize: 12,
            fontWeight: 600,
            padding: '2px 9px',
            borderRadius: 20,
          }}>
            {supportCount} need support
          </span>
        )}
        <span style={{
          marginLeft: 'auto',
          color: 'var(--slate-400)',
          fontSize: 13,
          transition: 'transform 0.2s',
          transform: collapsed ? 'rotate(-90deg)' : 'none',
        }}>▾</span>
      </div>

      {/* Candidates table */}
      {!collapsed && (
        <table style={{ width: '100%', borderCollapse: 'collapse' }}>
          <thead>
            <tr style={{ background: 'var(--slate-50)', borderBottom: '1px solid var(--slate-200)' }}>
              <Th>Candidate</Th>
              <Th>Role</Th>
              <Th center>Support Needed</Th>
              <Th width={220}>Assigned To</Th>
            </tr>
          </thead>
          <tbody>
            {candidates.map(c => (
              <CandidateRow
                key={c.id}
                candidate={c}
                assignment={assignments[c.id] || {}}
                teamMembers={teamMembers}
                onUpdate={(field, val) => onUpdate(c.id, field, val)}
              />
            ))}
          </tbody>
        </table>
      )}
    </div>
  )
}

function Th({ children, center, width }) {
  return (
    <th style={{
      padding: '9px 16px',
      textAlign: center ? 'center' : 'left',
      fontSize: 11,
      fontWeight: 700,
      color: 'var(--slate-500)',
      textTransform: 'uppercase',
      letterSpacing: '0.06em',
      width: width || 'auto',
    }}>
      {children}
    </th>
  )
}
