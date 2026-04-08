import { useState, useEffect, createContext, useContext } from 'react'
import { onAuth, signInWithGoogle } from '../services/firebase'
import { api } from '../services/api'
import '../styles/auth.css'

const AuthContext = createContext(null)

export function useAuth() {
  return useContext(AuthContext)
}

function GoogleIcon() {
  return (
    <svg viewBox="0 0 24 24" fill="none">
      <path d="M22.56 12.25c0-.78-.07-1.53-.2-2.25H12v4.26h5.92a5.06 5.06 0 01-2.2 3.32v2.77h3.57c2.08-1.92 3.28-4.74 3.28-8.1z" fill="#4285F4"/>
      <path d="M12 23c2.97 0 5.46-.98 7.28-2.66l-3.57-2.77c-.98.66-2.23 1.06-3.71 1.06-2.86 0-5.29-1.93-6.16-4.53H2.18v2.84C3.99 20.53 7.7 23 12 23z" fill="#34A853"/>
      <path d="M5.84 14.09c-.22-.66-.35-1.36-.35-2.09s.13-1.43.35-2.09V7.07H2.18C1.43 8.55 1 10.22 1 12s.43 3.45 1.18 4.93l2.85-2.22.81-.62z" fill="#FBBC05"/>
      <path d="M12 5.38c1.62 0 3.06.56 4.21 1.64l3.15-3.15C17.45 2.09 14.97 1 12 1 7.7 1 3.99 3.47 2.18 7.07l3.66 2.84c.87-2.6 3.3-4.53 6.16-4.53z" fill="#EA4335"/>
    </svg>
  )
}

function TrendingIcon() {
  return (
    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
      <polyline points="22 7 13.5 15.5 8.5 10.5 2 17"/>
      <polyline points="16 7 22 7 22 13"/>
    </svg>
  )
}

export default function AuthProvider({ children }) {
  const [firebaseUser, setFirebaseUser] = useState(null)
  const [profile, setProfile] = useState(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState(null)

  useEffect(() => {
    return onAuth((u) => {
      setFirebaseUser(u)
      if (!u) {
        setProfile(null)
        setLoading(false)
      }
    })
  }, [])

  useEffect(() => {
    if (!firebaseUser) return
    api.getMe()
      .then(setProfile)
      .catch(() => setProfile({ email: firebaseUser.email, is_admin: false }))
      .finally(() => setLoading(false))
  }, [firebaseUser])

  if (loading) {
    return <div className="auth-loading">Initializing</div>
  }

  if (!firebaseUser) {
    return (
      <div className="auth-screen">
        <div className="auth-card">
          <div className="auth-logo">
            <TrendingIcon />
          </div>
          <h1>Fin Cascade</h1>
          <p className="auth-subtitle">Real-time market intelligence &amp; cascade analysis</p>
          <div className="auth-divider">continue with</div>
          {error && <div className="auth-error">{error}</div>}
          <button className="auth-btn" onClick={async () => {
            try {
              setError(null)
              await signInWithGoogle()
            } catch (e) {
              setError(e.message)
            }
          }}>
            <GoogleIcon />
            Sign in with Google
          </button>
          <div className="auth-footer">
            Private dashboard &middot; Authorized access only
          </div>
        </div>
      </div>
    )
  }

  return (
    <AuthContext.Provider value={profile}>
      {children}
    </AuthContext.Provider>
  )
}
