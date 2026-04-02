import { useState, useEffect, useCallback, useRef } from 'react'
import Header from './components/Header'
import StageSection from './components/StageSection'
import SyncingBanner from './components/SyncingBanner'
import EmptyState from './components/EmptyState'

export default function App() {
  const [data, setData] = useState(null)
  const [assignments, setAssignments] = useState({})
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState(null)
  const [progressMsg, setProgressMsg] = useState('')
  const saveTimer = useRef(null)

  const load = useCallback(async (forceRefresh = false) => {
    setLoading(true)
    setError(null)
    setProgressMsg(forceRefresh ? 'Refreshing from Greenhouse…' : 'Loading…')

    try {
      // Poll until the backend has data (handles initial sync / 202 state)
      let candData = null
      while (true) {
        const res = await fetch(`/api/candidates${forceRefresh ? '?refresh=true' : ''}`)
        if (!res.ok) {
          const e = await res.json().catch(() => ({}))
          throw new Error(e.error || e.message || `HTTP ${res.status}`)
        }
        candData = await res.json()
        if (candData.syncing && !candData.candidates) {
          // Still syncing with no cached data yet — wait and retry
          setProgressMsg(candData.message || 'Syncing with Greenhouse…')
          await new Promise(r => setTimeout(r, 4000))
          forceRefresh = false // only force on first attempt
          continue
        }
        break
      }

      const assignData = await fetch('/api/assignments').then(r => r.json()).catch(() => ({}))
      setData(candData)
      setAssignments(assignData)
    } catch (err) {
      setError(err.message)
    } finally {
      setLoading(false)
      setProgressMsg('')
    }
  }, [])

  // SSE progress listener (only during explicit refresh)
  const listenProgress = useCallback(() => {
    const es = new EventSource('/api/progress')
    es.onmessage = (e) => {
      const { msg } = JSON.parse(e.data)
      if (msg === 'done' || msg === 'error') { es.close(); return }
      setProgressMsg(msg)
    }
    return es
  }, [])

  // Always listen to SSE progress so loading screen stays informative
  useEffect(() => {
    const es = new EventSource('/api/progress')
    es.onmessage = (e) => {
      const { msg } = JSON.parse(e.data)
      if (msg !== 'done' && msg !== 'error') setProgressMsg(msg)
    }
    return () => es.close()
  }, [])

  useEffect(() => { load() }, [load])

  const handleRefresh = useCallback(() => {
    const es = listenProgress()
    load(true).finally(() => es.close())
  }, [load, listenProgress])

  const saveAssignments = useCallback((next) => {
    clearTimeout(saveTimer.current)
    saveTimer.current = setTimeout(() => {
      fetch('/api/assignments', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(next),
      })
    }, 800)
  }, [])

  const updateAssignment = useCallback((appId, field, value) => {
    setAssignments(prev => {
      const next = { ...prev, [appId]: { ...(prev[appId] || {}), [field]: value } }
      saveAssignments(next)
      return next
    })
  }, [saveAssignments])

  const teamMembers = [...new Set(
    Object.values(assignments).map(a => a?.assignedTo).filter(Boolean)
  )].sort()

  if (loading) {
    return (
      <div style={{ minHeight: '100vh', background: 'var(--slate-100)' }}>
        <Header onRefresh={handleRefresh} refreshing={false} />
        <div style={{ display: 'flex', flexDirection: 'column', alignItems: 'center', justifyContent: 'center', height: '60vh', gap: 16 }}>
          <Spinner />
          <p style={{ color: 'var(--slate-500)', fontSize: 14 }}>{progressMsg || 'Loading…'}</p>
        </div>
      </div>
    )
  }

  if (error) {
    return (
      <div style={{ minHeight: '100vh', background: 'var(--slate-100)' }}>
        <Header onRefresh={handleRefresh} refreshing={false} />
        <div style={{ maxWidth: 800, margin: '40px auto', padding: '0 32px' }}>
          <div style={{ background: '#fef2f2', border: '1px solid #fecaca', borderRadius: 10, padding: 24, color: 'var(--red)' }}>
            <strong>Failed to load candidates</strong>
            <p style={{ marginTop: 8, fontSize: 14 }}>{error}</p>
          </div>
        </div>
      </div>
    )
  }

  const stages = data?.stageOrder || []
  const candidates = data?.candidates || {}
  const count = data?.count || 0

  return (
    <div style={{ minHeight: '100vh', background: 'var(--slate-100)' }}>
      <Header
        onRefresh={handleRefresh}
        refreshing={data?.refreshing}
        lastUpdated={data?.lastUpdated}
      />

      {data?.refreshing && <SyncingBanner message={progressMsg} />}

      <div style={{ maxWidth: 1300, margin: '0 auto', padding: '24px 32px' }}>
        {/* Summary */}
        <div style={{ display: 'flex', gap: 24, marginBottom: 20, fontSize: 14, color: 'var(--slate-500)' }}>
          <span><strong style={{ color: 'var(--slate-900)' }}>{count}</strong> active candidates</span>
          <span><strong style={{ color: 'var(--slate-900)' }}>{stages.length}</strong> stages</span>
          {(() => {
            const flagged = Object.values(assignments).filter(a => a?.supportNeeded).length
            return flagged > 0 ? (
              <span style={{ color: 'var(--orange)' }}>
                <strong>{flagged}</strong> flagged for support
              </span>
            ) : null
          })()}
        </div>

        {stages.length === 0 ? (
          <EmptyState />
        ) : (
          stages.map(stage => (
            <StageSection
              key={stage}
              stage={stage}
              candidates={candidates[stage] || []}
              assignments={assignments}
              teamMembers={teamMembers}
              onUpdate={updateAssignment}
            />
          ))
        )}
      </div>
    </div>
  )
}

function Spinner() {
  return (
    <div style={{
      width: 36, height: 36, borderRadius: '50%',
      border: '3px solid var(--slate-200)',
      borderTopColor: 'var(--indigo)',
      animation: 'spin 0.7s linear infinite',
    }} />
  )
}
