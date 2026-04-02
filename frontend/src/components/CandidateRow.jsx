import { useRef } from 'react'

export default function CandidateRow({ candidate, assignment, teamMembers, onUpdate }) {
  const needsSupport = !!assignment.supportNeeded
  const assignedTo = assignment.assignedTo || ''
  const datalistId = 'team-members-list'
  const rowRef = useRef(null)

  return (
    <tr
      ref={rowRef}
      style={{
        borderBottom: '1px solid var(--slate-100)',
        background: needsSupport ? '#fff7ed' : 'transparent',
        transition: 'background 0.1s',
      }}
      onMouseOver={e => {
        if (!needsSupport) e.currentTarget.style.background = 'var(--slate-50)'
      }}
      onMouseOut={e => {
        e.currentTarget.style.background = needsSupport ? '#fff7ed' : 'transparent'
      }}
    >
      {/* Candidate name + GH link */}
      <td style={{ padding: '11px 16px' }}>
        <a
          href={candidate.link}
          target="_blank"
          rel="noopener noreferrer"
          style={{
            color: 'var(--indigo-dark)',
            textDecoration: 'none',
            fontWeight: 600,
            fontSize: 14,
            display: 'inline-flex',
            alignItems: 'center',
            gap: 5,
          }}
          onMouseOver={e => e.currentTarget.style.textDecoration = 'underline'}
          onMouseOut={e => e.currentTarget.style.textDecoration = 'none'}
        >
          {candidate.name}
          <ExternalIcon />
        </a>
      </td>

      {/* Role */}
      <td style={{ padding: '11px 16px', fontSize: 13, color: 'var(--slate-500)' }}>
        {candidate.role}
      </td>

      {/* Support Needed toggle */}
      <td style={{ padding: '11px 16px', textAlign: 'center' }}>
        <div style={{ display: 'inline-flex', alignItems: 'center', gap: 8 }}>
          <Toggle
            checked={needsSupport}
            onChange={val => onUpdate('supportNeeded', val)}
          />
          {needsSupport && (
            <span style={{ fontSize: 11, fontWeight: 700, color: 'var(--orange)' }}>Needed</span>
          )}
        </div>
      </td>

      {/* Assigned To */}
      <td style={{ padding: '11px 16px' }}>
        <input
          list={datalistId}
          value={assignedTo}
          onChange={e => onUpdate('assignedTo', e.target.value)}
          placeholder="Assign team member…"
          style={{
            width: '100%',
            padding: '6px 10px',
            border: '1px solid var(--slate-200)',
            borderRadius: 7,
            fontSize: 13,
            color: 'var(--slate-900)',
            fontFamily: 'inherit',
            outline: 'none',
            transition: 'border-color 0.15s, box-shadow 0.15s',
          }}
          onFocus={e => {
            e.target.style.borderColor = 'var(--indigo)'
            e.target.style.boxShadow = '0 0 0 3px rgba(99,102,241,0.15)'
          }}
          onBlur={e => {
            e.target.style.borderColor = 'var(--slate-200)'
            e.target.style.boxShadow = 'none'
          }}
        />
        <datalist id={datalistId}>
          {teamMembers.map(name => <option key={name} value={name} />)}
        </datalist>
      </td>
    </tr>
  )
}

function Toggle({ checked, onChange }) {
  return (
    <button
      role="switch"
      aria-checked={checked}
      onClick={() => onChange(!checked)}
      style={{
        width: 38, height: 22,
        borderRadius: 11,
        border: 'none',
        background: checked ? 'var(--orange)' : 'var(--slate-200)',
        cursor: 'pointer',
        position: 'relative',
        transition: 'background 0.2s',
        flexShrink: 0,
        padding: 0,
      }}
    >
      <span style={{
        position: 'absolute',
        width: 16, height: 16,
        background: '#fff',
        borderRadius: '50%',
        top: 3,
        left: checked ? 19 : 3,
        transition: 'left 0.2s',
        boxShadow: '0 1px 3px rgba(0,0,0,0.2)',
      }} />
    </button>
  )
}

function ExternalIcon() {
  return (
    <svg width="11" height="11" viewBox="0 0 12 12" fill="none" style={{ opacity: 0.5, flexShrink: 0 }}>
      <path d="M2 2h8M10 2v8M10 2L4 8" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round" />
    </svg>
  )
}
